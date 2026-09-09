package logger

import (
	"errors"
	"syscall"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type syncErrorWriter struct {
	err error
}

func (w syncErrorWriter) Write(payload []byte) (int, error) {
	return len(payload), nil
}

func (w syncErrorWriter) Sync() error {
	return w.err
}

func TestSyncIgnoresUnsupportedTerminalSyncErrors(t *testing.T) {
	for _, syncErr := range []error{syscall.EINVAL, syscall.ENOTTY} {
		logger := zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			syncErrorWriter{err: syncErr},
			zapcore.InfoLevel,
		))

		if err := Sync(logger); err != nil {
			t.Fatalf("expected %v to be ignored, got %v", syncErr, err)
		}
	}
}

func TestSyncReturnsUnexpectedErrors(t *testing.T) {
	expected := errors.New("sync failed")
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		syncErrorWriter{err: expected},
		zapcore.InfoLevel,
	))

	if err := Sync(logger); !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
