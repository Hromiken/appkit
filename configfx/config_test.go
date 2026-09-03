package configfx

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testConfig struct {
	Postgres struct {
		DSN      string `koanf:"dsn"`
		MaxConns int32  `koanf:"max_conns"`
	} `koanf:"postgres"`
}

func TestLoadMergesFileAndEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	err := os.WriteFile(path, []byte("[postgres]\ndsn = \"from-file\"\nmax_conns = 4\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("TEST_POSTGRES__MAX_CONNS", "8")

	config, err := Load[testConfig](Options{
		FilePath:  path,
		EnvPrefix: "TEST_",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config.Postgres.DSN != "from-file" {
		t.Fatalf("DSN = %q, want from-file", config.Postgres.DSN)
	}
	if config.Postgres.MaxConns != 8 {
		t.Fatalf("MaxConns = %d, want 8", config.Postgres.MaxConns)
	}
}

type validatingConfig struct{}

func (*validatingConfig) Validate() error {
	return errors.New("invalid test config")
}

func TestLoadRunsValidation(t *testing.T) {
	_, err := Load[validatingConfig](Options{})
	if err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}

func TestLoadRequiredFile(t *testing.T) {
	_, err := Load[testConfig](Options{
		FilePath:     filepath.Join(t.TempDir(), "missing.toml"),
		FileRequired: true,
	})
	if err == nil {
		t.Fatal("Load() error = nil, want missing file error")
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("unknown = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Load[testConfig](Options{FilePath: path})
	if err == nil {
		t.Fatal("Load() error = nil, want unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("Load() error = %q, want unknown field context", err)
	}
}

func TestLoadCanAllowUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("unknown = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load[testConfig](Options{FilePath: path, AllowUnknownFields: true}); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadParsesDurationFromEnvironment(t *testing.T) {
	type durationConfig struct {
		Timeout time.Duration `koanf:"timeout"`
	}

	t.Setenv("TEST_TIMEOUT", "250ms")
	config, err := Load[durationConfig](Options{EnvPrefix: "TEST_"})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Timeout != 250*time.Millisecond {
		t.Fatalf("Timeout = %s, want 250ms", config.Timeout)
	}
}
