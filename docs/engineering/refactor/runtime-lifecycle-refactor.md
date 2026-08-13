# Torus Runtime Lifecycle Refactor — Engineering Record

> Historical engineering notes for the runtime lifecycle refactor.
>
> This document records **what changed, why it changed, what failed during the work, and how the final implementation was validated**. It is intentionally different from [`runtime-lifecycle.md`](../architecture/runtime-lifecycle.md), which documents the current architecture rather than the refactoring history.

---

## 1. Refactor Scope

The refactor hardened two related lifecycle boundaries:

1. **Configuration watcher lifecycle**
   - A cancelled watcher must not trigger a delayed reload.
   - Debounce state must be owned by the watcher lifecycle.

2. **Server/runtime lifecycle**
   - Runtime replacement must not race with shutdown.
   - Existing requests must continue using the runtime generation they acquired.
   - Rejected runtime generations must be cleaned up.
   - Shutdown must have explicit ownership and bounded completion semantics.
   - Startup and serving failures must leave the server in a truthful terminal state.

The resulting architecture separates three concerns:

```text
Configuration lifecycle
    ↓
Runtime generation lifecycle
    ↓
Server process lifecycle
```

---

# 2. Baseline

The refactor was performed on a dedicated branch:

```text
refactor/runtime-lifecycle
```

The initial verification established a clean baseline:

```text
go test ./...
go test -race ./...
```

Both passed before lifecycle changes were introduced.

This baseline mattered because the goal was to distinguish:

- pre-existing failures,
- regressions introduced by the refactor,
- and bugs exposed by new lifecycle tests.

---

# 3. Phase 1 — Make Configuration Watching Shutdown-Safe

## 3.1 The original bug

The watcher originally used `time.AfterFunc()` for debounce:

```text
fsnotify event
    ↓
time.AfterFunc()
    ↓
detached callback
    ↓
Reload()
```

The timer callback was detached from the watcher event loop.

That allowed:

```text
configuration change
    ↓
debounce callback scheduled
    ↓
shutdown / context cancellation
    ↓
watcher exits
    ↓
timer fires
    ↓
Reload()
```

This violated the intended lifecycle rule:

> Once the watcher has stopped, it must not initiate new reload work.

---

## 3.2 First design correction

The watcher was changed to use a narrow interface:

```go
type ReloadManager interface {
    Reload() error
}
```

instead of depending directly on `*reload.Manager`.

This was initially implemented incorrectly with a concrete type assertion inside `New()`:

```go
manager.( *reload.Manager )
```

That defeated the abstraction and caused the fake test manager to panic.

The fix was simple:

> Store the `ReloadManager` interface directly.

This also made the watcher unit-testable without depending on the concrete reload manager.

---

## 3.3 Event-loop-owned debounce

The final debounce design uses:

```go
var debounce *time.Timer
var debounceC <-chan time.Time
```

The watcher event loop owns the timer.

Configuration event:

```text
fsnotify
   ↓
stop previous timer
   ↓
start new timer
```

Timer expiration:

```text
debounce.C
   ↓
main select
   ↓
Reload()
```

There is no detached reload callback.

A nil `debounceC` disables the timer case in the `select`, making the event loop the sole owner of debounce state.

---

## 3.4 Shutdown behavior

When the watcher context is cancelled:

```text
ctx.Done()
    ↓
stop pending debounce
    ↓
return from Start()
```

The resulting ownership boundary is:

```text
Watcher
 ├── filesystem events
 ├── debounce
 ├── reload initiation
 └── termination

Reload Manager
 └── runtime construction

Server
 └── runtime installation
```

---

## 3.5 Phase 1 regression tests

### `TestWatcherDoesNotReloadAfterShutdown`

Protects the original failure mode:

```text
watcher starts
    ↓
config event
    ↓
debounce pending
    ↓
cancel context
    ↓
watcher exits
    ↓
reload count must remain 0
```

### `TestWatcherReloadsAfterConfigurationChange`

Protects normal hot reload behavior:

```text
config change
    ↓
fsnotify
    ↓
debounce
    ↓
Reload()
    ↓
reload count == 1
```

The tests intentionally use the real filesystem watcher instead of introducing a large fsnotify abstraction. A small startup sleep remains only to allow the real filesystem watch to initialize.

---

# 4. Phase 2 — Introduce an Explicit Server Lifecycle

## 4.1 The original weakness

Before the refactor, the server had readiness state:

```go
ready atomic.Bool
```

but no authoritative lifecycle state.

That meant:

```text
Shutdown()
    ↓
ready = false
```

did not automatically prevent:

```text
Reload()
    ↓
new runtime installed
```

The system needed to distinguish:

- readiness,
- lifecycle state,
- runtime ownership.

---

## 4.2 New state machine

The server gained:

```go
type serverState uint32

const (
    serverStarting serverState = iota
    serverRunning
    serverShuttingDown
    serverStopped
)
```

Normal lifecycle:

```text
STARTING
    ↓
RUNNING
    ↓
SHUTTING_DOWN
    ↓
STOPPED
```

Startup failures are terminal:

```text
STARTING
    ↓
failure
    ↓
STOPPED
```

---

# 5. Phase 3 — Harden Runtime Reload

## 5.1 Reload contract

`Server.Reload()` now returns an error and is accepted only in `serverRunning`.

```text
STARTING       → reject
RUNNING        → accept
SHUTTING_DOWN  → reject
STOPPED        → reject
```

The explicit error is:

```go
ErrServerNotRunning
```

---

## 5.2 Runtime swap boundary

The lifecycle-sensitive portion is serialized under `runtimeMu`:

```text
runtimeMu.Lock()
    ↓
check state
    ↓
swap runtime
    ↓
runtimeMu.Unlock()
```

Only after the swap is the old runtime retired.

```text
Runtime N
   ↓
swap
   ↓
Runtime N+1 active
   ↓
Runtime N.Stop()
```

---

## 5.3 Why `Runtime.Stop()` stays outside the mutex

`Runtime.Stop()` can block on:

```go
requestWG.Wait()
workerWG.Wait()
```

Holding `runtimeMu` while draining can deadlock with a request that needs `runtimeMu.RLock()` in order to acquire its runtime.

The final rule is:

> `runtimeMu` protects ownership transitions, not long-running destruction.

---

# 6. Phase 4 — Make Runtime Ownership Explicit

`BuildRuntime()` starts health workers.

Therefore building a runtime creates resources before the runtime is necessarily installed into the server.

The ownership transition is:

```text
BuildRuntime()
    ↓
creator owns runtime
    ↓
Server.Reload() succeeds
    ↓
server owns runtime
```

If installation fails:

```text
BuildRuntime()
    ↓
Server.Reload()
    ↓
rejected
    ↓
creator stops runtime
```

`reload.Manager.Reload()` now performs:

```go
rt.Stop()
```

when `Server.Reload()` rejects the newly built runtime.

This prevents health-worker leaks.

---

# 7. Phase 5 — Harden Shutdown

## 7.1 Shutdown runtime ownership

Shutdown captures the currently active runtime at the moment the server transitions into `SHUTTING_DOWN`.

```text
RUNNING
    ↓
runtimeMu.Lock()
    ↓
SHUTTING_DOWN
    ↓
capture active runtime
    ↓
runtimeMu.Unlock()
```

The shutdown path never performs a late runtime lookup after the drain has begun.

This establishes:

> Shutdown owns the runtime active at the `RUNNING → SHUTTING_DOWN` transition.

---

## 7.2 Readiness transition

Shutdown immediately changes:

```text
state = SHUTTING_DOWN
ready = false
```

before waiting for existing requests.

This is important because readiness and request draining serve different purposes:

```text
ready = false
    ↓
stop advertising availability

HTTP drain
    ↓
allow already-active work to finish
```

A dedicated regression test verifies that readiness is cleared **while the request is still draining**, not only after shutdown completes.

---

## 7.3 Graceful shutdown

The shutdown algorithm is:

```text
RUNNING
    ↓
SHUTTING_DOWN
    ↓
ready = false
    ↓
capture runtime
    ↓
http.Server.Shutdown(timeout)
    ↓
Runtime.Stop()
    ↓
STOPPED
```

`http.Server.Shutdown()` provides the HTTP-level draining behavior.

`Runtime.Stop()` provides runtime-level cleanup and waits for runtime-owned work.

---

# 8. Concurrent Shutdown Ownership

The first caller to transition the server into `SHUTTING_DOWN` becomes the shutdown owner.

Additional callers do not start a second shutdown operation.

They wait on:

```go
shutdownDone chan struct{}
```

and receive:

```go
shutdownErr error
```

Conceptually:

```text
Shutdown A
    ↓
RUNNING → SHUTTING_DOWN
    ↓
perform shutdown
    ↓
store result
    ↓
close(shutdownDone)
           ↑
           │
Shutdown B
    ↓
wait
    ↓
same result
```

This makes shutdown idempotent at the lifecycle level.

A call after `STOPPED` returns the stored result.

---

# 9. Forced Cancellation

Graceful shutdown has a bounded deadline.

If the deadline expires:

```text
graceful shutdown timeout
    ↓
forceCancel()
    ↓
second shutdown window
    ↓
Runtime.Stop()
```

The forced path is intentionally different from normal draining.

```text
Graceful drain
    = allow current work to finish

Forced cancellation
    = stop waiting indefinitely and cancel current work
```

---

## 9.1 End-to-end cancellation path

The final forced-cancellation test validates the complete chain:

```text
shutdown deadline
    ↓
forceCancel()
    ↓
request context cancelled
    ↓
ReverseProxy / upstream request exits
    ↓
proxy request terminates
    ↓
Runtime.Stop()
    ↓
STOPPED
```

The `context canceled` proxy error emitted by the test is expected because the test deliberately causes request cancellation.

---

## 9.2 Forced-shutdown failure

If the forced shutdown attempt itself fails:

```text
Shutdown()
    ↓
error
```

the server remains:

```text
SHUTTING_DOWN
```

rather than falsely claiming:

```text
STOPPED
```

The state is intentionally truthful about the server's actual condition.

---

# 10. Startup Failure Handling

Two startup/serving failures were explicitly tested.

## Listener creation failure

```text
STARTING
    ↓
net.Listen() fails
    ↓
STOPPED
```

The already-created runtime is also stopped because it may already own health workers.

## Unexpected `Serve()` failure

```text
RUNNING
    ↓
Serve() returns unexpected error
    ↓
ready = false
    ↓
cancel outstanding request contexts
    ↓
stop active runtime
    ↓
STOPPED
```

`http.ErrServerClosed` remains a normal path because it is expected during graceful shutdown.

---

# 11. Concurrent Reload Audit

Two concurrent reloads are allowed at the server boundary.

Example:

```text
Reload A → Runtime 2
Reload B → Runtime 3
```

`runtimeMu` serializes the actual swaps.

The final runtime may legitimately be Runtime 2 or Runtime 3 depending on which caller acquires the mutex last.

The important properties are:

- Runtime 1 cannot remain active.
- The final runtime is one of the successfully installed generations.
- Replaced runtimes are retired.
- The active runtime pointer is never concurrently mutated without synchronization.

Normal filesystem-driven reloads are already serialized by the watcher event loop, so this is primarily defensive server-level behavior.

---

# 12. Request / Runtime Synchronization

One of the most important concurrency issues discovered during the work was the ordering between:

```text
runtime.Load()
```

and:

```text
runtime.AcquireRequest()
```

The final protocol is:

```text
runtimeMu.RLock()
    ↓
runtime.Load()
    ↓
runtime.AcquireRequest()
    ↓
runtimeMu.RUnlock()
```

Doing this instead:

```text
runtime.Load()
    ↓
runtime.AcquireRequest()
```

without synchronization would allow reload to swap and retire the runtime in the gap.

The protected invariant is:

> Once a request acquires a runtime, that runtime cannot be retired until the request releases it.

---

# 13. Test Strategy Developed During the Refactor

The refactor intentionally targets concrete production failure modes rather than attempting to prove every scheduler interleaving.

Coverage includes:

### Configuration watcher

```text
TestWatcherDoesNotReloadAfterShutdown
TestWatcherReloadsAfterConfigurationChange
```

### Runtime reload

```text
TestServerReload_RuntimeSwap
TestServerReload_WaitsForActiveRequests
TestServerReload_AfterShutdownBegins
TestServerReload_AfterStopped
TestServerReload_ConcurrentReloads
```

### Shutdown

```text
TestServerShutdown_AfterReloadStopsNewRuntime
TestServerShutdown_ConcurrentCallsWaitForSameShutdown
TestServerShutdown_MarksUnreadyBeforeDrain
```

### Graceful / forced shutdown

```text
TestGracefulShutdown
TestGracefulShutdown_ForceCancellation
```

### Startup failures

```text
TestServerStart_ListenerFailureStopsServer
TestServerStart_UnexpectedServeFailureStopsServer
```

### Reload-manager ownership

```text
TestManagerReload_StopsRuntimeWhenServerRejectsReload
```

The tests generally synchronize on real events instead of arbitrary sleep durations. For example, active-request tests wait until the backend confirms that the request has actually started before triggering reload or shutdown.

---

# 14. Verification Performed

The final repository-wide verification was:

```bash
go test ./...
go test -race ./...
go vet ./...
```

The lifecycle-specific stress suite was:

```bash
go test -race ./internal/proxy -count=50 -shuffle=on
```

All passed.

The lifecycle tests were also repeatedly run individually with the race detector during development.

---

# 15. What Changed from the Original Design

## Watcher

Before:

```text
fsnotify
    ↓
detached AfterFunc
    ↓
Reload()
```

After:

```text
fsnotify
    ↓
event-loop-owned timer
    ↓
Reload()
```

Result:

> Watcher termination also terminates ownership of pending reload work.

---

## Server lifecycle

Before:

```text
readiness state
+
reload/shutdown without an explicit lifecycle boundary
```

After:

```text
STARTING
    ↓
RUNNING
    ↓
SHUTTING_DOWN
    ↓
STOPPED
```

with `runtimeMu` defining the synchronization boundary for lifecycle-sensitive runtime ownership.

---

# 16. Resulting Invariants

The refactor established these practical invariants:

1. A watcher cannot initiate a reload after it has stopped.
2. Only a `RUNNING` server may install a new runtime.
3. A request remains associated with the runtime generation it acquired.
4. A successfully replaced runtime is eventually stopped.
5. A rejected runtime is stopped by its creator.
6. Shutdown owns the runtime captured at shutdown start.
7. `runtimeMu` is never held while performing blocking drains.
8. Only one shutdown caller owns the shutdown operation.
9. Startup failures transition the server to `STOPPED`.
10. Readiness becomes false before shutdown draining finishes.
11. Forced cancellation propagates into an in-flight proxied request.
12. A failed forced shutdown does not falsely transition to `STOPPED`.

---

# 17. Final Architecture After the Refactor

The final runtime lifecycle can be summarized as:

```text
                 CONFIGURATION LIFECYCLE

Config change
     ↓
ConfigWatcher
     ↓
debounce
     ↓
ReloadManager
     ↓
BuildRuntime(N+1)
     ↓
Server.Reload()
     ↓
publish N+1
     ↓
Runtime N drains
     ↓
Runtime N retires
```

And:

```text
                 SERVER LIFECYCLE

STARTING
   │
   ├── startup failure ──→ STOPPED
   │
   ▼
RUNNING
   │
   ├── reload ──→ new runtime generation
   │
   └── shutdown
          │
          ▼
     SHUTTING_DOWN
          │
          ├── ready = false
          ├── reject reload
          ├── drain HTTP requests
          │
          ├── graceful success
          │      ↓
          │   Runtime.Stop()
          │      ↓
          │   STOPPED
          │
          └── timeout
                 ↓
             forceCancel()
                 ↓
             second shutdown
                 ↓
             Runtime.Stop()
                 ↓
             STOPPED
```

---

# 18. Final Design Assessment

The refactor did not attempt to redesign Torus around a multi-process architecture.

Instead, it strengthened the existing single-process generation model by making its lifecycle boundaries explicit:

```text
runtime generation
      +
ownership
      +
publication
      +
retirement
      +
server lifecycle
      +
bounded shutdown
```

The important architectural lesson from the work is:

> **The difficult part of live configuration replacement is not swapping configuration data. It is defining who owns each generation, when ownership changes, and exactly when the previous generation is safe to retire.**

That ownership model became the foundation for the current Torus runtime lifecycle.
