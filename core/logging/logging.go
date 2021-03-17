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
// By convention, this should appear in the same .go file as the package docstring:
//  var logger = logging.New("Foo")
func New(pkg string) *zap.Logger {
	lvl := parseLevel(GetLevel(pkg))
	return root.Named(pkg).
		WithOptions(zap.IncreaseLevel(zap.NewAtomicLevelAt(lvl)))
}

func parseLevel(lvl rune) zapcore.Level {
	switch lvl {
	case 'V', 'D':
		return zapcore.DebugLevel
	case 'I':
		return zapcore.InfoLevel
	case 'W':
		return zapcore.WarnLevel
	case 'E':
		return zapcore.ErrorLevel
	case 'F', 'N':
		return zapcore.DPanicLevel
	}
	return zapcore.InfoLevel
}

// GetLevel returns configured log level of a package as a letter.
func GetLevel(pkg string) rune {
	lvl, ok := os.LookupEnv("NDNDPDK_LOG_" + pkg)
	if !ok {
		lvl, ok = os.LookupEnv("NDNDPDK_LOG")
	}
	if !ok || len(lvl) == 0 {
		return 0
	}
	return rune(lvl[0])
}
