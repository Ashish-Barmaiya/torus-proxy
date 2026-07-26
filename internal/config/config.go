package config

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const CurrentAPIVersion = "v1"

type Config struct {
	APIVersion  string            `yaml:"apiVersion"`
	Server      ServerConfig      `yaml:"server"`
	HealthCheck HealthCheckConfig `yaml:"health"`
	Routes      []RouteConfig     `yaml:"routes"`
	Tls         *TlsConfig        `yaml:"tls"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type HealthCheckConfig struct {
	IntervalMs int    `yaml:"interval_ms"`
	TimeoutMs  int    `yaml:"timeout_ms"`
	Path       string `yaml:"path"`
}

type RouteConfig struct {
	Path      string   `yaml:"path"`
	Upstreams []string `yaml:"upstream"`
}

type TlsConfig struct {
	CertFile   string `yaml:"cert_file"`
	KeyFile    string `yaml:"key_file"`
	MinVersion string `yaml:"min_version,omitempty"` // defaults to TLS 1.2 if not specified
}

func (h HealthCheckConfig) Interval() time.Duration {
	return time.Duration(h.IntervalMs) * time.Millisecond
}

func (h HealthCheckConfig) Timeout() time.Duration {
	return time.Duration(h.TimeoutMs) * time.Millisecond
}

// Validate config file
func (c *Config) Validate() error {
	// Server
	if strings.TrimSpace(c.Server.Addr) == "" {
		return fmt.Errorf("server.addr must not be empty")
	}

	// Health checks
	if c.HealthCheck.IntervalMs <= 0 {
		return fmt.Errorf("health.interval_ms must be greater than 0")
	}

	if c.HealthCheck.TimeoutMs <= 0 {
		return fmt.Errorf("health.timeout_ms must be greater than 0")
	}

	if c.HealthCheck.TimeoutMs > c.HealthCheck.IntervalMs {
		return fmt.Errorf("health.timeout_ms cannot be greater than health.interval_ms")
	}

	if strings.TrimSpace(c.HealthCheck.Path) == "" {
		return fmt.Errorf("health.path must not be empty")
	}

	if !strings.HasPrefix(c.HealthCheck.Path, "/") {
		return fmt.Errorf("health.path must start with '/'")
	}

	// Routes
	if len(c.Routes) == 0 {
		return fmt.Errorf("at least one route must be configured")
	}

	seenPaths := make(map[string]struct{})

	for i, route := range c.Routes {
		if strings.TrimSpace(route.Path) == "" {
			return fmt.Errorf("routes[%d].path must not be empty", i)
		}

		if _, exists := seenPaths[route.Path]; exists {
			return fmt.Errorf("duplicate route path %q", route.Path)
		}
		seenPaths[route.Path] = struct{}{}

		if len(route.Upstreams) == 0 {
			return fmt.Errorf("routes[%d] must contain at least one upstream", i)
		}

		for j, upstream := range route.Upstreams {
			if strings.TrimSpace(upstream) == "" {
				return fmt.Errorf("routes[%d].upstream[%d] must not be empty", i, j)
			}

			u, err := url.ParseRequestURI(upstream)
			if err != nil {
				return fmt.Errorf(
					"routes[%d].upstream[%d]: invalid URL: %w",
					i,
					j,
					err,
				)
			}

			switch u.Scheme {
			case "http", "https":
			default:
				return fmt.Errorf(
					"routes[%d].upstream[%d]: unsupported URL scheme %q",
					i,
					j,
					u.Scheme,
				)
			}

			if u.Host == "" {
				return fmt.Errorf(
					"routes[%d].upstream[%d]: missing host",
					i,
					j,
				)
			}
		}
	}

	// TLS
	if c.Tls != nil {
		if strings.TrimSpace(c.Tls.CertFile) == "" {
			return fmt.Errorf("tls.cert_file must not be empty")
		}

		if strings.TrimSpace(c.Tls.KeyFile) == "" {
			return fmt.Errorf("tls.key_file must not be empty")
		}

		switch c.Tls.MinVersion {
		case "", "1.2", "1.3":
		default:
			return fmt.Errorf(
				"tls.min_version must be one of: 1.2, 1.3",
			)
		}
	}

	return nil
}

func (t *TlsConfig) LoadTlsConfig() (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(t.CertFile, t.KeyFile)
	if err != nil {
		return nil, err
	}

	var minVersion uint16
	switch t.MinVersion {
	case "", "1.2":
		minVersion = tls.VersionTLS12
	case "1.3":
		minVersion = tls.VersionTLS13
	default:
		return nil, fmt.Errorf("unsupported or invalid TLS version: %s (only 1.2 and 1.3 are allowed)", t.MinVersion)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   minVersion,
	}, nil
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Parse the YAML file into a Config struct
	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}

	// Validate the configuration schema version
	switch config.APIVersion {
	case CurrentAPIVersion:
		// Supported configuration schema

	default:
		return nil, fmt.Errorf(
			"unsupported config api version %q (expected %q)",
			config.APIVersion,
			CurrentAPIVersion,
		)
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}
