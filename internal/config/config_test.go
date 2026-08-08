package config

import (
	"testing"
)

func validConfig() Config {
	return Config{
		Server: ServerConfig{
			Addr: ":8080",
		},
		HealthCheck: HealthCheckConfig{
			IntervalMs: 5000,
			TimeoutMs:  1000,
			Path:       "/health",
		},
		Services: []ServiceConfig{
			{
				Name: "api",
				Upstreams: []string{
					"http://localhost:3001",
					"http://localhost:3002",
				},
			},
		},
		Routes: []RouteConfig{
			{
				Path:    "/",
				Service: "api",
			},
		},
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     validConfig(),
			wantErr: false,
		},
		{
			name: "empty server addr",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Server = ServerConfig{}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "no services",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "no routes",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Routes = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "empty service name",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Name = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "duplicate service names",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services = append(
					cfg.Services,
					ServiceConfig{
						Name: "api",
						Upstreams: []string{
							"http://localhost:4001",
						},
					},
				)
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "service without upstreams",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = nil
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "empty upstream",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{""}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid upstream url",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{
					"http://localhost:",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "unsupported upstream scheme",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{
					"ftp://localhost:3001",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "upstream missing host",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{
					"http://:3001",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "upstream missing port",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{
					"http://localhost",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "invalid upstream port",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{
					"http://localhost:abc",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "upstream port out of range",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Services[0].Upstreams = []string{
					"http://localhost:65536",
				}
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "duplicate routes",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Routes = append(
					cfg.Routes,
					RouteConfig{
						Path:    "/",
						Service: "api",
					},
				)
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "route without service",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Routes[0].Service = ""
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "route references nonexistent service",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Routes[0].Service = "missing"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "health timeout greater than interval",
			cfg: func() Config {
				cfg := validConfig()
				cfg.HealthCheck.IntervalMs = 1000
				cfg.HealthCheck.TimeoutMs = 5000
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "health path missing slash",
			cfg: func() Config {
				cfg := validConfig()
				cfg.HealthCheck.Path = "health"
				return cfg
			}(),
			wantErr: true,
		},
		{
			name: "tls config valid",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Tls = &TlsConfig{
					CertFile:   "cert.pem",
					KeyFile:    "key.pem",
					MinVersion: "1.3",
				}
				return cfg
			}(),
			wantErr: false,
		},
		{
			name: "invalid tls version",
			cfg: func() Config {
				cfg := validConfig()
				cfg.Tls = &TlsConfig{
					CertFile:   "cert.pem",
					KeyFile:    "key.pem",
					MinVersion: "2.0",
				}
				return cfg
			}(),
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
