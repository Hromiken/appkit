package healthfx

import (
	"errors"
	"time"
)

const defaultCheckTimeout = 5 * time.Second

// Config controls readiness checks.
type Config struct {
	// CheckTimeout limits the total duration of a readiness request. A zero
	// value uses a safe default of five seconds.
	CheckTimeout time.Duration `koanf:"check_timeout"`
}

func (config Config) withDefaults() Config {
	if config.CheckTimeout == 0 {
		config.CheckTimeout = defaultCheckTimeout
	}
	return config
}

// Validate rejects negative timeout values.
func (config Config) Validate() error {
	if config.CheckTimeout < 0 {
		return errors.New("health check timeout must not be negative")
	}
	return nil
}
