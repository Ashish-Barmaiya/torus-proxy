# Torus Runtime Lifecycle

> Architecture reference for the Torus reverse proxy runtime, request lifecycle, configuration reload, readiness, graceful shutdown, forced cancellation, and lifecycle failure handling.

## 1. Overview

Torus treats the active proxy configuration as a **runtime generation**, rather than mutable global state.

A runtime contains the routing and upstream state required to serve traffic for one configuration generation. When the configuration changes, Torus builds a new runtime, installs it as the active runtime, and retires the previous runtime only after work associated with that runtime has drained.

The process itself has a separate lifecycle:

```text
STARTING
   |
   | successful startup
   v
RUNNING
   |
   | shutdown begins
   v
SHUTTING_DOWN
   |
   | shutdown completes
   v
STOPPED
```

These two lifecycles interact:

```text
Configuration lifecycle:

Configuration change
        |
        v
    Build Runtime N+1
        |
        v
    Install Runtime N+1
        |
        v
    Retire Runtime N
```

```text
Process lifecycle:

STARTING -> RUNNING -> SHUTTING_DOWN -> STOPPED
```

The key architectural property is that **runtime replacement and process shutdown are coordinated explicitly**. Once shutdown begins, no new runtime can be installed.

---

## 2. Runtime Architecture

A `Runtime` represents one complete proxy configuration generation.

Conceptually:

```text
Runtime
├── Generation
├── Router
├── Services
│   └── Backend pools
├── TLS configuration
├── Runtime cancellation context
├── Health-check workers
└── In-flight request accounting
```

The runtime generation is monotonically increasing:

```text
Runtime 1
   |
   | configuration reload
   v
Runtime 2
   |
   | configuration reload
   v
Runtime 3
```

The generation is useful for logging, metrics, debugging, and identifying which configuration generation is active.

### Runtime ownership

Runtime ownership follows a simple rule:

> The server owns a runtime only after successful installation.

Before installation, the component that constructed the runtime owns it.

Therefore:

```text
 BuildRuntime()
      |
      v
  Runtime N+1
      |
      v
Server.Reload()
   /          \
success      rejected
  |              |
  v              v
Server        Creator
owns          cleans up
runtime          |
                 v
              Runtime.Stop()
```

This prevents background health workers from leaking when a runtime is built successfully but cannot be installed because the server is no longer running.

---

## 3. Process Startup

The process startup path begins in `main.go`.

High-level sequence:

```text
main()
  |
  +-- create logger
  |
  +-- parse configuration path
  |
  +-- create reload.Manager
  |
  +-- BuildInitialRuntime()
  |
  +-- create root context
  |
  +-- create proxy.Server
  |
  +-- attach Server to ReloadManager
  |
  +-- start Server
  |
  +-- start ConfigWatcher
  |
  v
RUNNING
```

Initial runtime construction happens before the HTTP server begins listening.

A server created by `NewServer()` starts in `serverStarting`.

A successful `Start()` performs the server initialization required to accept traffic and eventually transitions the state to `serverRunning`.

### Startup failure

Startup failures are terminal for the `Server` instance.

If listener creation fails:

```text
STARTING
   |
   | net.Listen() fails
   v
STOPPED
```

The already-created runtime is also stopped because it may already own background health workers.

An unexpected `http.Server.Serve()` failure follows the same principle:

```text
RUNNING
   |
   | unexpected Serve() failure
   v
STOPPED
```

The server becomes unready, cancels outstanding request contexts, and stops the active runtime.

`http.ErrServerClosed` is different: it is the expected termination result produced by graceful shutdown and is therefore not treated as an unexpected serving failure.

---

## 4. Runtime Construction

`BuildRuntime()` constructs a complete runtime from the validated configuration.

The construction pipeline is:

```text
Config
  |
  +-- create Router
  |
  +-- create runtime cancellation context
  |
  +-- load TLS configuration
  |
  +-- allocate generation
  |
  +-- create Backends
  |
  +-- start health-prober workers
  |
  +-- create Services
  |
  +-- register Routes
  |
  v
Runtime
```

Runtime construction is not just allocation of configuration data. Health-check workers are started during construction.

This is why ownership must be explicit even for runtimes that are never installed.

---

## 5. Active Runtime

The server stores its current runtime in an atomic pointer.

At any moment while the server is running, one runtime generation is considered active:

```text
Server
  |
  +--> active Runtime N
```

A successful reload changes that relationship:

```text
Before:

Server -> Runtime N

After:

Server -> Runtime N+1
```

The previous runtime remains alive long enough for its existing work to complete and is then retired with `Runtime.Stop()`.

---

## 6. Request Lifecycle

Each request must first associate itself with the runtime generation it will use.

The critical sequence is:

```text
HTTP request
    |
    v
runtimeMu.RLock()
    |
    +-- runtime.Load()
    |
    +-- runtime.AcquireRequest()
    |
    v
runtimeMu.RUnlock()
    |
    v
route request
    |
    v
select backend
    |
    v
ReverseProxy
    |
    v
runtime.ReleaseRequest()
```

The request's runtime is acquired under the read lock because runtime replacement can happen concurrently.

The important ordering is:

```text
Load runtime
    +
Acquire request
```

Both must occur before the read lock is released.

Without that synchronization, a reload could swap and stop the runtime after `Load()` but before `AcquireRequest()`.

### Request/runtime invariant

> Once a request acquires a runtime, that runtime remains alive until the request releases it.

This allows old runtime generations to drain safely during reload.

---

## 7. Runtime Draining

A runtime tracks two categories of work.

### In-flight requests

Requests are tracked with:

```go
requestWG sync.WaitGroup
```

A request calls:

```go
AcquireRequest()
```

when it starts using the runtime and:

```go
ReleaseRequest()
```

when it completes.

### Background workers

Background workers are tracked with:

```go
workerWG sync.WaitGroup
```

Workers register with:

```go
AddWorker()
```

and deregister with:

```go
DoneWorker()
```

### Runtime.Stop()

`Runtime.Stop()` is the runtime-level drain operation:

```text
Runtime.Stop()
      |
      +-- cancel runtime context
      |
      +-- wait for in-flight requests
      |
      +-- wait for background workers
      |
      v
runtime fully retired
```

A runtime is not fully retired simply because it is no longer the active runtime. Its associated work must also drain.

---

## 8. Configuration Watching

Torus watches the configuration file using `fsnotify`.

The current watcher architecture is event-loop owned:

```text
Configuration file change
        |
        v
     fsnotify
        |
        v
   watcher event loop
        |
        v
   debounce timer
        |
        v
      Reload()
```

The debounce timer is owned by the watcher event loop rather than a detached callback.

Conceptually:

```text
fsnotify event
    |
    +-- stop existing timer
    |
    +-- start new timer
    |
    v
 debounce.C
    |
    v
 main select
    |
    v
Reload()
```

A nil debounce channel disables the timer case in the event loop when there is no pending debounce.

### Watcher shutdown

When the watcher context is cancelled:

```text
ctx.Done()
   |
   v
stop pending debounce
   |
   v
exit watcher
```

The watcher therefore owns:

- filesystem event processing
- debounce state
- reload initiation
- its own termination

A cancelled watcher cannot initiate a late reload from a detached timer callback.

---

## 9. Runtime Reload Lifecycle

The complete hot-reload path is:

```text
Configuration change
        |
        v
ConfigWatcher
        |
      debounce
        |
        v
ReloadManager.Reload()
        |
        v
LoadConfig()
        |
        v
BuildRuntime()
        |
        v
Server.Reload(newRuntime)
        |
        v
runtimeMu.Lock()
        |
        v
state == RUNNING?
      /      \
    yes       no
     |         |
     v         v
   swap      reject
     |         |
     v         v
 unlock     newRuntime.Stop()
     |
     v
oldRuntime.Stop()
```

### Reload eligibility

A reload is accepted only while the server is `serverRunning`.

```text
STARTING       -> reject
RUNNING        -> accept
SHUTTING_DOWN  -> reject
STOPPED        -> reject
```

When rejected, `Server.Reload()` returns `ErrServerNotRunning`.

---

## 10. Runtime Replacement and Existing Requests

Suppose Runtime 1 is active and a reload creates Runtime 2.

Existing requests and new requests are deliberately separated:

```text
                  Runtime 1
                     |
          existing request(s)
                     |
                     v
                still running

Reload
   |
   v
Runtime 2 becomes active
   |
   +-- new requests -> Runtime 2
```

Runtime 1 is retired only after its active requests and background workers have drained.

This is the core zero-disruption property of runtime replacement:

> New traffic moves to the new runtime while in-flight work finishes on the old runtime.

---

## 11. `runtimeMu` as the Lifecycle Boundary

`runtimeMu` is a `sync.RWMutex` with two complementary roles.

### Read side

The read side protects request acquisition:

```text
RLock
  |
  +-- Load runtime
  +-- Acquire request
  |
RUnlock
```

### Write side

The write side protects lifecycle-sensitive operations:

- checking whether reload is allowed
- swapping the active runtime
- transitioning to `SHUTTING_DOWN`
- capturing the shutdown-owned runtime

The mutex therefore establishes the ordering boundary between request ownership and runtime lifecycle transitions.

### Why the mutex is not held during drains

`Runtime.Stop()` may wait indefinitely within its configured lifecycle bounds for requests and workers to complete.

Holding `runtimeMu` while waiting could deadlock with a request that needs `runtimeMu.RLock()` to acquire the runtime reference.

Therefore:

```text
runtimeMu
  |
  +-- state check
  +-- runtime swap / snapshot
  |
  +-- unlock

then:

blocking runtime/request drain
```

The mutex protects **ownership transitions**, not the drain itself.

---

## 12. Readiness

Readiness and lifecycle state are separate concepts.

### Readiness

`ready` answers:

> Should an orchestrator consider this server available to receive traffic?

### Lifecycle state

`state` answers:

> What lifecycle operations are currently allowed?

At the beginning of shutdown:

```text
state = SHUTTING_DOWN
ready = false
```

The readiness transition happens before the drain completes.

```text
RUNNING
   |
   v
SHUTTING_DOWN
   |
   +-- ready = false
   |
   +-- existing requests still drain
```

This prevents the server from continuing to advertise readiness while it is already shutting down.

---

## 13. Signal Handling

The process-level signal path is:

```text
SIGINT / SIGTERM
        |
        v
signal.NotifyContext()
        |
        v
signal context cancelled
        |
        v
root context cancelled
        |
        +-----------------> ConfigWatcher exits
        |
        v
Server.Shutdown()
```

The signal path and the server drain are related but have different responsibilities.

The root context coordinates application-level cancellation.

`Server.Shutdown()` performs the HTTP/runtime lifecycle transition and drain.

---

## 14. Graceful Shutdown Lifecycle

Shutdown begins by establishing a lifecycle boundary under `runtimeMu`:

```text
runtimeMu.Lock()
     |
     v
RUNNING -> SHUTTING_DOWN
     |
     +-- ready = false
     +-- capture active runtime
     |
     v
runtimeMu.Unlock()
```

The HTTP server is then asked to shut down gracefully using the configured timeout.

Conceptually:

```text
http.Server.Shutdown(timeout)
        |
        +-- stop accepting new connections
        |
        +-- allow existing HTTP work to drain
        |
        v
     complete
```

Once the HTTP server has finished draining:

```text
Runtime.Stop()
      |
      v
STOPPED
```

### Shutdown ownership

Shutdown captures the runtime that was active when the `RUNNING -> SHUTTING_DOWN` transition occurred.

That runtime is the one shutdown is responsible for retiring.

No later runtime replacement is allowed after that transition.

---

## 15. Concurrent Shutdown

Shutdown is explicitly single-owner.

The first caller that transitions the server to `SHUTTING_DOWN` performs the shutdown.

A concurrent second caller waits for the first operation:

```text
Shutdown A
    |
    v
RUNNING -> SHUTTING_DOWN
    |
    +-- perform shutdown
    |
    +-- store result
    |
    +-- close(shutdownDone)
             ^
             |
Shutdown B --+
    |
    +-- wait
    |
    +-- return same result
```

This prevents multiple callers from independently owning the shutdown operation.

A shutdown called after `STOPPED` returns the stored shutdown result.

---

## 16. Forced Cancellation

Graceful shutdown is bounded by a deadline because the proxy cannot safely wait forever for arbitrary downstream work.

The fallback is:

```text
Graceful timeout
       |
       v
forceCancel()
       |
       v
request contexts cancelled
       |
       v
ReverseProxy / upstream request exits
       |
       v
second shutdown window
       |
       v
Runtime.Stop()
       |
       v
STOPPED
```

The intended semantic is:

> Graceful shutdown allows existing work to finish; forced cancellation stops waiting indefinitely when the grace period is exhausted.

The forced path is deliberately bounded by a second shutdown window.

### Context propagation

The server creates a base request context using `context.WithCancel()` and supplies it through `http.Server.BaseContext`.

The hierarchy is conceptually:

```text
baseCtx
   |
   +-- Request A
   |
   +-- Request B
   |
   +-- Request C
```

Cancelling the base context therefore propagates to derived request contexts.

For proxied requests, that cancellation reaches the upstream request through Go's HTTP request context propagation.

The forced-cancellation test verifies the complete chain rather than merely checking that `forceCancel()` was invoked.

---

## 17. Shutdown Failure Semantics

A successful graceful shutdown reaches `STOPPED`.

A graceful timeout followed by successful forced cancellation also reaches `STOPPED`.

If the forced shutdown itself fails, the server does **not** claim successful termination:

```text
SHUTTING_DOWN
      |
      | forced shutdown fails
      v
SHUTTING_DOWN + error
```

This preserves the distinction between:

- shutdown completed
- shutdown attempted but did not complete successfully

Concurrent shutdown callers receive the same stored error.

---

## 18. Failure Scenarios

| Scenario | Result |
|---|---|
| Initial runtime construction fails | Process exits before server startup |
| Listener creation fails | `STARTING -> STOPPED`, runtime cleaned up |
| Unexpected `Serve()` failure | `RUNNING -> STOPPED`, runtime cleaned up |
| Reload while `RUNNING` | New runtime installed |
| Reload while `SHUTTING_DOWN` | Rejected with `ErrServerNotRunning` |
| Reload while `STOPPED` | Rejected with `ErrServerNotRunning` |
| Runtime built but installation rejected | Newly built runtime stopped by creator |
| Graceful shutdown completes | `STOPPED` |
| Graceful timeout + forced cancellation succeeds | `STOPPED` |
| Forced shutdown fails | `SHUTTING_DOWN` + error |
| Concurrent shutdown | Secondary caller waits for primary shutdown |

---

## 19. Complete Runtime Lifecycle

The complete lifecycle can be viewed as two interacting state machines.

### Process lifecycle

```text
                         +----------------+
                         |    STARTING    |
                         +-------+--------+
                                 |
                         successful Start
                                 |
                                 v
                         +----------------+
                         |    RUNNING     |
                         +-------+--------+
                                 |
                           Shutdown()
                                 |
                                 v
                         +----------------+
                         | SHUTTING_DOWN  |
                         +-------+--------+
                                 |
                         shutdown complete
                                 |
                                 v
                         +----------------+
                         |    STOPPED     |
                         +----------------+
```

### Runtime lifecycle

```text
Runtime N
    |
    | configuration change
    v
Build Runtime N+1
    |
    v
Install Runtime N+1
    |
    v
Runtime N remains only for its
existing requests/workers
    |
    v
Runtime N.Stop()
    |
    v
Runtime N retired
```

### Combined model

```text
                        PROCESS

 STARTING --------------------------------------------------+
    |                                                       |
    | successful start                                      |
    v                                                       | startup failure
 RUNNING                                                    |
    |                                                       v
    |                                                +-------------+
    |                                                |   STOPPED   |
    |                                                +-------------+
    |
    +------ configuration change
    |                  |
    |                  v
    |           Build Runtime N+1
    |                  |
    |             Server.Reload()
    |                  |
    |            +-----+------+
    |            |            |
    |         accept        reject
    |            |            |
    |            v            v
    |       swap runtime   Stop N+1
    |            |
    |            v
    |       Stop Runtime N
    |
    +------ SIGTERM / SIGINT
                       |
                       v
                SHUTTING_DOWN
                       |
                       +-- ready = false
                       +-- reject reload
                       +-- drain HTTP
                       +-- forced cancel if needed
                       +-- Runtime.Stop()
                       |
                       v
                    STOPPED
```

---

## 20. Lifecycle Invariants

The runtime architecture depends on these invariants:

1. **Only `RUNNING` servers accept runtime installation.**
2. **A request that acquires Runtime N keeps Runtime N alive until the request completes.**
3. **New requests acquire the currently active runtime generation.**
4. **A successfully replaced runtime is eventually stopped.**
5. **A rejected runtime is stopped by the component that constructed it.**
6. **Shutdown owns the runtime active at the moment the server enters `SHUTTING_DOWN`.**
7. **`runtimeMu` is never held while waiting for requests or workers to drain.**
8. **Only one concurrent caller owns the shutdown operation.**
9. **Readiness becomes false when shutdown begins, before the drain completes.**
10. **Forced cancellation propagates into in-flight request contexts when the grace period expires.**
11. **Startup and unexpected serving failures leave the server in a terminal `STOPPED` state.**

These invariants are more important than the individual implementation details because they define the behavior that must remain true as Torus evolves.

---

## 21. Failure Ownership

The lifecycle is easiest to reason about when ownership is explicit.

### During runtime construction

```text
ReloadManager
     |
     +-- builds Runtime N+1
```

### After successful installation

```text
Server
     |
     +-- owns Runtime N+1
```

### After runtime replacement

```text
Server
     |
     +-- owns Runtime N+1
     |
     +-- Runtime N is being retired
```

### During shutdown

```text
Server.Shutdown()
     |
     +-- owns shutdown operation
     +-- owns runtime captured at shutdown start
```

### Concurrent shutdown caller

```text
Secondary caller
     |
     +-- does not own shutdown
     +-- waits for shutdownDone
```

Explicit ownership prevents two components from attempting to clean up the same lifecycle resource independently.

---

## 22. Testing the Lifecycle

The lifecycle is tested through concrete behavioral contracts rather than an attempt to enumerate every possible scheduler interleaving.

### Configuration watcher

- `TestWatcherDoesNotReloadAfterShutdown`
- `TestWatcherReloadsAfterConfigurationChange`

### Runtime reload

- `TestServerReload_RuntimeSwap`
- `TestServerReload_WaitsForActiveRequests`
- `TestServerReload_AfterShutdownBegins`
- `TestServerReload_AfterStopped`
- `TestServerReload_ConcurrentReloads`

### Shutdown

- `TestServerShutdown_AfterReloadStopsNewRuntime`
- `TestServerShutdown_ConcurrentCallsWaitForSameShutdown`
- `TestServerShutdown_MarksUnreadyBeforeDrain`
- `TestGracefulShutdown`
- `TestGracefulShutdown_ForceCancellation`

### Startup failure

- `TestServerStart_ListenerFailureStopsServer`
- `TestServerStart_UnexpectedServeFailureStopsServer`

### Runtime ownership

- `TestManagerReload_StopsRuntimeWhenServerRejectsReload`

The tests are intentionally synchronized around real lifecycle events such as request arrival, shutdown state transitions, and upstream cancellation rather than relying on arbitrary timing wherever practical.

---

## 23. Final Mental Model

If the implementation details are forgotten, remember three rules.

### Rule 1 — Configuration creates generations

```text
Configuration
     |
     v
Runtime N+1
     |
     v
swap into server
     |
     v
retire Runtime N
```

### Rule 2 — Requests belong to the runtime they acquired

```text
Request
  |
  +--> Runtime N
         |
         +--> remains alive until request completes
```

### Rule 3 — Shutdown creates a hard boundary

```text
RUNNING
   |
   | shutdown begins
   v
SHUTTING_DOWN
   |
   +-- no new runtime installation
   +-- readiness false
   +-- drain existing work
   +-- cancel if deadline expires
   |
   v
STOPPED
```

Together, these rules describe the runtime lifecycle of Torus.

## Design Trade-offs

Torus uses an in-process, generation-based runtime model. This simplifies configuration replacement and keeps listener ownership stable, but it deliberately trades away some capabilities provided by process-generation architectures.

### Advantages

- **No process handoff:** Runtime reloads do not require process replacement, IPC, PID coordination, or listener FD transfer.
- **Explicit ownership:** Each runtime owns its resources and can be independently drained and retired.
- **Atomic replacement:** New configuration is built before the active runtime is replaced, so failed construction leaves the current runtime unaffected.
- **Clear request semantics:** Existing requests remain associated with the runtime generation they acquired.
- **Simple listener lifecycle:** The HTTP listener remains owned by the same server across configuration reloads.

### Costs

- **Shared address space:** All runtime generations exist within the same process, so a process-level failure affects the entire proxy.
- **Temporary resource overlap:** During reload, old and new runtime generations may coexist, increasing memory and other resource usage.
- **Explicit lifecycle management:** Runtime-owned resources must be correctly classified and retired; missing an ownership boundary can cause leaks.
- **No binary hot restart:** The architecture supports runtime/configuration replacement, not replacement of the running executable without process restart.
- **Long-lived work can delay retirement:** Connections, streams, or other long-running operations can keep an old runtime alive until they complete or are cancelled.

Torus therefore prioritizes **simple in-process configuration replacement, explicit runtime ownership, and predictable request draining** over process isolation and seamless binary replacement.
