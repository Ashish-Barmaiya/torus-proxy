package configwatcher

import (
	"context"
	"log/slog"
	"path/filepath"
	"torus-proxy/internal/proxy"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	configPath string
	logger     *slog.Logger
	server     *proxy.Server
}

func New(
	configPath string,
	logger *slog.Logger,
	server *proxy.Server,
) *Watcher {
	return &Watcher{
		configPath: configPath,
		logger:     logger,
		server:     server,
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

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

			w.logger.Info(
				"configuration changed",
				"name", event.Name,
				"op", event.Op.String(),
			)

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
