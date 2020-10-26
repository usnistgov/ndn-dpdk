// Package logger is a thin wrapper of logrus library.
package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// New creates a logger.
func New(pkg string) logrus.FieldLogger {
	return NewWithPrefix(pkg, pkg)
}

// NewWithPrefix creates a logger with specified prefix.
func NewWithPrefix(pkg, prefix string) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(parseLevel(pkg))

	formatter := &prefixFormatter{}
	formatter.prefix = fmt.Sprintf("[%s] ", prefix)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = time.StampMicro
	logger.Formatter = formatter

	return logger
}

// GetLevel returns configured log level of a package as a letter.
func GetLevel(pkg string) rune {
	lvl, ok := os.LookupEnv("NDNDPDK_LOG_" + pkg)
	if !ok {
		lvl, ok = os.LookupEnv("NDNDPDK_LOG")
	}
	if !ok || len(lvl) == 0 {
		lvl = "I"
	}
	return rune(lvl[0])
}

func parseLevel(pkg string) logrus.Level {
	lvl := GetLevel(pkg)
	switch lvl {
	case 'V', 'D':
		return logrus.DebugLevel
	case 'I':
		return logrus.InfoLevel
	case 'W':
		return logrus.WarnLevel
	case 'E':
		return logrus.ErrorLevel
	case 'F':
		return logrus.FatalLevel
	case 'N':
		return logrus.PanicLevel
	}
	return logrus.InfoLevel
}

type prefixFormatter struct {
	logrus.TextFormatter
	prefix string
}

func (f *prefixFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	entry.Message = f.prefix + entry.Message
	return f.TextFormatter.Format(entry)
}
