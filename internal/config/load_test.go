package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestObservabilityDefaultsToEnabled(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := `
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

services:
  - name: api
    upstreams:
      - http://localhost:3001

routes:
  - path: /
    service: api
`

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.Observability.EnabledValue() {
		t.Fatal("expected observability to default to enabled when the setting is omitted")
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

services:
  - name: api
    upstreams:
      - http://localhost:3001

routes:
  - path: /
    service: api
`,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			content: `
apiVersion: v2

server:
  addr: :

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

services:
  - name: api
    upstreams:
      - http://localhost:3001

routes:
  - path: /
    service: api
`,
			wantErr: true,
		},
		{
			name: "unsupported api version",
			content: `
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

services:
  - name: api
    upstreams:
      - http://localhost:3001

routes:
  - path: /
    service: api
`,
			wantErr: true,
		},
		{
			name: "semantic validation failure",
			content: `
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

services:
  - name: api
    upstreams:
      - http://localhost:

routes:
  - path: /
    service: api
`,
			wantErr: true,
		},
		{
			name: "route references missing service",
			content: `
apiVersion: v2

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

services:
  - name: api
    upstreams:
      - http://localhost:3001

routes:
  - path: /
    service: missing
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")

			if err := os.WriteFile(configPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}

			_, err := LoadConfig(configPath)

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
