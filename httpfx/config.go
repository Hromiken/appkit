package httpfx

import (
	"errors"
	"time"
)

// Config controls the shared HTTP server.
type Config struct {
	Address           string        `koanf:"address"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	ReadTimeout       time.Duration `koanf:"read_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	IdleTimeout       time.Duration `koanf:"idle_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`
}

func (config Config) withDefaults() Config {
	if config.Address == "" {
		config.Address = ":8080"
	}
	if config.ReadHeaderTimeout == 0 {
		config.ReadHeaderTimeout = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 15 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 60 * time.Second
	}
	if config.ShutdownTimeout == 0 {
		config.ShutdownTimeout = 10 * time.Second
	}
	return config
}

// Validate rejects negative timeout values.
func (config Config) Validate() error {
	if config.ReadHeaderTimeout < 0 ||
		config.ReadTimeout < 0 ||
		config.WriteTimeout < 0 ||
		config.IdleTimeout < 0 ||
		config.ShutdownTimeout < 0 {
		return errors.New("HTTP timeouts must not be negative")
	}
	return nil
}
