package loggingfx

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewAppliesLevel(t *testing.T) {
	logger, err := New(Config{Level: "debug"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if !logger.Core().Enabled(zapcore.DebugLevel) {
		t.Fatal("debug logging is disabled")
	}
}

func TestNewRejectsInvalidLevel(t *testing.T) {
	if _, err := New(Config{Level: "everything"}); err == nil {
		t.Fatal("New() error = nil, want invalid level error")
	}
}

func TestLoggerLifecycleSyncsLogger(t *testing.T) {
	writer := &testWriteSyncer{}
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), writer, zapcore.InfoLevel))
	lifecycle := fxtest.NewLifecycle(t)
	registerLifecycle(lifecycle, logger)

	lifecycle.RequireStart()
	lifecycle.RequireStop()
	if !writer.synced {
		t.Fatal("logger was not synced on stop")
	}
}

func TestLoggerLifecycleReturnsSyncError(t *testing.T) {
	wantErr := errors.New("sync failed")
	writer := &testWriteSyncer{syncErr: wantErr}
	logger := zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), writer, zapcore.InfoLevel))
	lifecycle := fxtest.NewLifecycle(t)
	registerLifecycle(lifecycle, logger)

	lifecycle.RequireStart()
	if err := lifecycle.Stop(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("Lifecycle.Stop() error = %v, want %v", err, wantErr)
	}
}

type testWriteSyncer struct {
	synced  bool
	syncErr error
}

func (writer *testWriteSyncer) Write(data []byte) (int, error) {
	return len(data), nil
}

func (writer *testWriteSyncer) Sync() error {
	writer.synced = true
	return writer.syncErr
}
