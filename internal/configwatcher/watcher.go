package configwatcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"
	"torus-proxy/internal/reload"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	configPath string
	logger     *slog.Logger
	manager    *reload.Manager
}

func New(configPath string, logger *slog.Logger, manager *reload.Manager) *Watcher {
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
			debounce = time.AfterFunc(250*time.Millisecond, func() {
				if err := w.manager.Reload(); err != nil {
					w.logger.Error(
						"failed to reload configuration",
						"error", err,
					)
				}
			})

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
