package logger

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

type BaltoModule string

const (
	BALTO_ROUTER          BaltoModule = "router"
	BALTO_PROXY           BaltoModule = "proxy"
	BALTO_HEALTHCHECKER   BaltoModule = "healthchecker"
	BALTO_BALANCER        BaltoModule = "balancer"
	BALTO_BACKENDPOOL     BaltoModule = "backendpool"
	BALTO_CIRCUIT_BREAKER BaltoModule = "circuit-breaker"
	BALTO_SERVER          BaltoModule = "server"
)

type Config struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

var globalLogger zerolog.Logger

func Init(cfg *Config) {
	if cfg == nil {
		cfg = &Config{
			Level: "info",
			Path:  "",
		}
	}

	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	var output io.Writer = os.Stdout
	if cfg.Path != "" {
		file, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			output = file
		}
	}

	globalLogger = zerolog.New(output).With().Timestamp().Logger()
}

func Info(module BaltoModule, msg string, fields ...interface{}) {
	event := globalLogger.Info()
	event.Str("module", string(module))
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fields[i].(string)
			value := fields[i+1]
			switch v := value.(type) {
			case string:
				event.Str(key, v)
			case int:
				event.Int(key, v)
			case bool:
				event.Bool(key, v)
			default:
				event.Interface(key, v)
			}
		}
	}
	event.Msg(msg)
}

func Warn(module BaltoModule, msg string, fields ...interface{}) {
	event := globalLogger.Warn()
	event.Str("module", string(module))
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fields[i].(string)
			value := fields[i+1]
			switch v := value.(type) {
			case string:
				event.Str(key, v)
			case int:
				event.Int(key, v)
			case bool:
				event.Bool(key, v)
			default:
				event.Interface(key, v)
			}
		}
	}
	event.Msg(msg)
}

func Error(module BaltoModule, msg string, fields ...interface{}) {
	event := globalLogger.Error()
	event.Str("module", string(module))
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fields[i].(string)
			value := fields[i+1]
			switch v := value.(type) {
			case string:
				event.Str(key, v)
			case int:
				event.Int(key, v)
			case bool:
				event.Bool(key, v)
			default:
				event.Interface(key, v)
			}
		}
	}
	event.Msg(msg)
}

func Debug(module BaltoModule, msg string, fields ...interface{}) {
	event := globalLogger.Debug()
	event.Str("module", string(module))
	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			key := fields[i].(string)
			value := fields[i+1]
			switch v := value.(type) {
			case string:
				event.Str(key, v)
			case int:
				event.Int(key, v)
			case bool:
				event.Bool(key, v)
			default:
				event.Interface(key, v)
			}
		}
	}
	event.Msg(msg)
}
