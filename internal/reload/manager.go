package reload

import (
	"fmt"
	"log/slog"
	"torus-proxy/internal/config"
	"torus-proxy/internal/observability"
	"torus-proxy/internal/proxy"
	"torus-proxy/internal/runtime"
)

type Manager struct {
	configPath string
	logger     *slog.Logger
	server     *proxy.Server
}

func NewManager(configPath string, logger *slog.Logger) *Manager {
	return &Manager{
		configPath: configPath,
		logger:     logger,
	}
}

func (m *Manager) SetServer(server *proxy.Server) {
	m.server = server
}

// Helper
func (m *Manager) buildRuntime() (*runtime.Runtime, error) {
	cfg, err := config.LoadConfig(m.configPath)
	if err != nil {
		return nil, err
	}

	return runtime.BuildRuntime(cfg, m.logger)
}

func (m *Manager) BuildInitialRuntime() (*runtime.Runtime, error) {
	rt, err := m.buildRuntime()
	if err != nil {
		return nil, err
	}

	observability.SetRuntimeGeneration(rt.Generation)

	return rt, nil
}

func (m *Manager) Reload() error {
	if m.server == nil {
		return fmt.Errorf("reload manager has no server")
	}
	rt, err := m.buildRuntime()
	if err != nil {
		observability.RecordRuntimeReload(false)
		return err
	}

	m.server.Reload(rt)
	return nil
}
