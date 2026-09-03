// Package configfx loads typed application configuration for Fx.
package configfx

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/toml/v2"
	envprovider "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
)

const delimiter = "."

// Validator can be implemented by an application config to validate itself
// after all sources have been merged.
type Validator interface {
	Validate() error
}

// Options controls configuration loading. Values from the environment override
// values loaded from the TOML file.
type Options struct {
	FilePath     string
	FileRequired bool
	EnvPrefix    string
	EnvSeparator string
	// AllowUnknownFields disables strict decoding. By default, unknown keys
	// are rejected to catch configuration typos during startup.
	AllowUnknownFields bool
}

// DefaultOptions returns conventional local-development settings.
//
// Environment variables use a double underscore for nesting:
// APP_POSTGRES__DSN maps to postgres.dsn.
func DefaultOptions() Options {
	return Options{
		FilePath:     "config.toml",
		EnvPrefix:    "APP_",
		EnvSeparator: "__",
	}
}

// Load reads TOML and environment configuration into T.
func Load[T any](options Options) (*T, error) {
	k := koanf.New(delimiter)

	if options.FilePath != "" {
		err := k.Load(file.Provider(options.FilePath), toml.Parser())
		if err != nil && (options.FileRequired || !errors.Is(err, os.ErrNotExist)) {
			return nil, fmt.Errorf("load config file %q: %w", options.FilePath, err)
		}
	}

	if options.EnvPrefix != "" {
		separator := options.EnvSeparator
		if separator == "" {
			separator = "__"
		}

		provider := envprovider.Provider(delimiter, envprovider.Opt{
			Prefix: options.EnvPrefix,
			TransformFunc: func(key, value string) (string, any) {
				key = strings.TrimPrefix(key, options.EnvPrefix)
				key = strings.ToLower(key)
				key = strings.ReplaceAll(key, separator, delimiter)
				return key, value
			},
		})

		if err := k.Load(provider, nil); err != nil {
			return nil, fmt.Errorf("load environment config: %w", err)
		}
	}

	var config T
	decoderConfig := &mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.TextUnmarshallerHookFunc(),
		),
		ErrorUnused:      !options.AllowUnknownFields,
		WeaklyTypedInput: true,
	}
	if err := k.UnmarshalWithConf("", &config, koanf.UnmarshalConf{DecoderConfig: decoderConfig}); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if validator, ok := any(&config).(Validator); ok {
		if err := validator.Validate(); err != nil {
			return nil, fmt.Errorf("validate config: %w", err)
		}
	}

	return &config, nil
}

// Provide returns an Fx option that provides a single typed config instance.
func Provide[T any](options Options) fx.Option {
	return fx.Provide(func() (*T, error) {
		return Load[T](options)
	})
}
