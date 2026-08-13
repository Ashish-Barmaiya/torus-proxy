package configwatcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type fakeReloadManager struct {
	mu          sync.Mutex
	reloadCount int
	reloaded    chan struct{}
}

func newFakeReloadManager() *fakeReloadManager {
	return &fakeReloadManager{
		reloaded: make(chan struct{}, 10),
	}
}

func (m *fakeReloadManager) Reload() error {
	m.mu.Lock()
	m.reloadCount++
	m.mu.Unlock()

	m.reloaded <- struct{}{}

	return nil
}

func (m *fakeReloadManager) ReloadCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.reloadCount
}

func TestWatcherDoesNotReloadAfterShutdown(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "torus.yaml")

	if err := os.WriteFile(configPath, []byte("initial: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	manager := newFakeReloadManager()

	logger := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}),
	)

	watcher := New(configPath, logger, manager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- watcher.Start(ctx)
	}()

	// Give the watcher time to establish the filesystem watch
	time.Sleep(100 * time.Millisecond)

	// Trigger a configuration event
	if err := os.WriteFile(configPath, []byte("initial: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Shutdown before the 250ms debounce expires
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watcher returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after cancellation")
	}

	// Give the pending debounce enough time to fire if it wasn't cancelled.
	time.Sleep(350 * time.Millisecond)

	if got := manager.ReloadCount(); got != 0 {
		t.Fatalf(
			"expected no reload after shutdown, got %d reload(s)",
			got,
		)
	}
}

func TestWatcherReloadsAfterConfigurationChange(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "torus.yaml")

	if err := os.WriteFile(configPath, []byte("initial: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	manager := newFakeReloadManager()

	logger := slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelError,
		}),
	)

	watcher := New(configPath, logger, manager)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)

	go func() {
		done <- watcher.Start(ctx)
	}()

	// Give the watcher time to establish the filesystem watch
	time.Sleep(100 * time.Millisecond)

	// Trigger a configuration change.
	if err := os.WriteFile(configPath, []byte("initial: false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// The watcher should debounce the event and then reload
	select {
	case <-manager.reloaded:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("expected configuration reload, but none occurred")
	}

	if got := manager.ReloadCount(); got != 1 {
		t.Fatalf("expected exactly 1 reload, got %d", got)
	}

	// Cleanly stop the watcher
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watcher returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after cancellation")
	}
}
