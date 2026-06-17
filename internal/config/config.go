package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	HealthCheck HealthCheckConfig `yaml:"health"`
	Routes      []RouteConfig     `yaml:"routes"`
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

func (h HealthCheckConfig) Interval() time.Duration {
	return time.Duration(h.IntervalMs) * time.Millisecond
}

func (h HealthCheckConfig) Timeout() time.Duration {
	return time.Duration(h.TimeoutMs) * time.Millisecond
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
