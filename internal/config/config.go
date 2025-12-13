package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	__BALTO_CONFIG_PATH       string = "configs/balto.config.yaml"
	__BALTO_DEFAULT_LOGS_PATH string = "/var/log/balto/balto.log"
	__BALTO_DEFAULT_PORT      string = ":5500"
	__BALTO_DEFAULT_ALGO      string = "round-robbin"
)

type Config struct {
	Global   GlobalConfig    `yaml:"global"`
	Server   ServerConfig    `yaml:"server"`
	Services []ServiceConfig `yaml:"services"`
}

type GlobalConfig struct {
	LoadBalancing LoadBalancingConfig `yaml:"load_balancing"`
	TLS           TLSConfig           `yaml:"tls"`
	Logging       LoggingConfig       `yaml:"logging"`
	Metrics       MetricsConfig       `yaml:"metrics"`
	CORS          CORSConfig          `yaml:"cors"`
	Timeouts      TimeoutsConfig      `yaml:"timeouts"`
}

type LoadBalancingConfig struct {
	Algorithm string `yaml:"algorithm"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type TimeoutsConfig struct {
	Read  string `yaml:"read"`
	Write string `yaml:"write"`
	Idle  string `yaml:"idle"`
}

type ServerConfig struct {
	Port string `yaml:"port"`
}

type ServiceConfig struct {
	Domain     string   `yaml:"domain"`
	PathPrefix string   `yaml:"path_prefix"`
	Ports      []string `yaml:"ports"`
}

func Default() *Config {
	return &Config{
		Global: GlobalConfig{
			LoadBalancing: LoadBalancingConfig{
				Algorithm: __BALTO_DEFAULT_ALGO,
			},
			TLS: TLSConfig{
				Enabled:  false,
				CertFile: "",
				KeyFile:  "",
			},
			Logging: LoggingConfig{
				Level: "info",
				Path:  __BALTO_DEFAULT_LOGS_PATH,
			},
			Metrics: MetricsConfig{
				Enabled: true,
			},
			CORS: CORSConfig{
				Enabled:        false,
				AllowedOrigins: []string{},
			},
			Timeouts: TimeoutsConfig{
				Read:  "5s",
				Write: "5s",
				Idle:  "30s",
			},
		},
		Server: ServerConfig{
			Port: __BALTO_DEFAULT_PORT,
		},
		Services: []ServiceConfig{},
	}
}

func Load() (*Config, error) {
	data, err := os.ReadFile(__BALTO_CONFIG_PATH)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

func (t *TimeoutsConfig) ReadDuration() (time.Duration, error) {
	if t.Read == "" {
		return 5 * time.Second, nil
	}
	return time.ParseDuration(t.Read)
}

func (t *TimeoutsConfig) WriteDuration() (time.Duration, error) {
	if t.Write == "" {
		return 5 * time.Second, nil
	}
	return time.ParseDuration(t.Write)
}

func (t *TimeoutsConfig) IdleDuration() (time.Duration, error) {
	if t.Idle == "" {
		return 30 * time.Second, nil
	}
	return time.ParseDuration(t.Idle)
}

func (s *ServerConfig) Address() string {
	if s.Port[0] != ':' {
		return ":" + s.Port
	}
	return s.Port
}
