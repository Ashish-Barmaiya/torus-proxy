package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `
apiVersion: v1

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

routes:
  - path: /
    upstream:
      - http://localhost:3001
`,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			content: `
apiVersion: v1

server:
  addr: :
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

routes:
  - path: /
    upstream:
      - http://localhost:3001
`,
			wantErr: true,
		},
		{
			name: "semantic validation failure",
			content: `
apiVersion: v1

server:
  addr: ":8080"

health:
  interval_ms: 5000
  timeout_ms: 1000
  path: /health

routes:
  - path: /
    upstream:
      - http://localhost:
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
