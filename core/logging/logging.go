// Package logging is a thin wrapper of zap logging library.
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var root = func() *zap.Logger {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		os.Stderr,
		zap.DebugLevel,
	)
	return zap.New(core)
}()

// Named creates a named logger without initialization.
func Named(pkg string) *zap.Logger {
	return root.Named(pkg)
}

// New creates a logger initialized with configured log level.
//
// By NDN-DPDK codebase convention, this should appear in the same .go file as the package docstring:
//  var logger = logging.New("Foo")
func New(pkg string) *zap.Logger {
	return Named(pkg).WithOptions(zap.IncreaseLevel(GetLevel(pkg).al))
}
