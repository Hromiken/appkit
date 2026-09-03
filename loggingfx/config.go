package loggingfx

// Config controls the application logger.
type Config struct {
	Development bool              `koanf:"development"`
	Level       string            `koanf:"level"`
	Encoding    string            `koanf:"encoding"`
	Service     string            `koanf:"service"`
	Environment string            `koanf:"environment"`
	Fields      map[string]string `koanf:"fields"`
}
