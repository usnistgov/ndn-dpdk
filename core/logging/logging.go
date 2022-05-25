// Package logging is a thin wrapper of zap logging library.
package logging

import (
	"log"
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

// StdLogger creates a log.Logger that logs to zap.Logger at specified level.
func StdLogger(logger *zap.Logger, lvl zapcore.Level) *log.Logger {
	return log.New(&stdLoggerWriter{
		logger: logger,
		lvl:    lvl,
	}, "", 0)
}

type stdLoggerWriter struct {
	logger *zap.Logger
	lvl    zapcore.Level
}

func (w *stdLoggerWriter) Write(p []byte) (n int, e error) {
	n = len(p)
	if n > 0 && p[n-1] == '\n' {
		p = p[:n-1]
	}
	w.logger.Check(w.lvl, string(p)).Write()
	return
}
