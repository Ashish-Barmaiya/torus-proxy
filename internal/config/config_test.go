package config

import (
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
						Upstreams: []string{
							"http://localhost:3001",
							"http://localhost:3002",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty server addr",
			cfg: Config{
				Server: ServerConfig{},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path:      "/",
						Upstreams: []string{"http://localhost:3001"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no routes",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate routes",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path:      "/",
						Upstreams: []string{"http://localhost:3001"},
					},
					{
						Path:      "/",
						Upstreams: []string{"http://localhost:3002"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "route without upstream",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid upstream url",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
						Upstreams: []string{
							"http://localhost:",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "health timeout greater than interval",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 1000,
					TimeoutMs:  5000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
						Upstreams: []string{
							"http://localhost:3001",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "health path missing slash",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
						Upstreams: []string{
							"http://localhost:3001",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "tls config valid",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
						Upstreams: []string{
							"http://localhost:3001",
						},
					},
				},
				Tls: &TlsConfig{
					CertFile:   "cert.pem",
					KeyFile:    "key.pem",
					MinVersion: "1.3",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tls version",
			cfg: Config{
				Server: ServerConfig{
					Addr: ":8080",
				},
				HealthCheck: HealthCheckConfig{
					IntervalMs: 5000,
					TimeoutMs:  1000,
					Path:       "/health",
				},
				Routes: []RouteConfig{
					{
						Path: "/",
						Upstreams: []string{
							"http://localhost:3001",
						},
					},
				},
				Tls: &TlsConfig{
					CertFile:   "cert.pem",
					KeyFile:    "key.pem",
					MinVersion: "2.0",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
