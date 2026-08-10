package configwatcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"
	"torus-proxy/internal/observability"

	"github.com/fsnotify/fsnotify"
)

type ReloadManager interface {
	Reload() error
}

type Watcher struct {
	configPath string
	logger     *slog.Logger
	manager    ReloadManager
}

func New(configPath string, logger *slog.Logger, manager ReloadManager) *Watcher {
	return &Watcher{
		configPath: configPath,
		logger:     logger,
		manager:    manager,
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	var debounce *time.Timer
	var debounceC <-chan time.Time

	configPath, err := filepath.Abs(w.configPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(w.configPath)

	if err := watcher.Add(dir); err != nil {
		return err
	}

	w.logger.Info(
		"configuration watcher started",
		"directory", dir,
	)

	for {
		select {
		case <-ctx.Done():
			if debounce != nil {
				debounce.Stop()
			}
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			eventPath, err := filepath.Abs(event.Name)
			if err != nil {
				continue
			}

			if eventPath != configPath {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.NewTimer(250 * time.Millisecond)
			debounceC = debounce.C

		case <-debounceC:
			debounceC = nil

			if err := w.manager.Reload(); err != nil {
				observability.RecordRuntimeReload(false)
				w.logger.Error(
					"failed to reload configuration",
					"error", err,
				)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}

			w.logger.Error(
				"configuration watcher error",
				"error", err,
			)
		}
	}
}
