package config

import (
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
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
	return &config, nil
}
