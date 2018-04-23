package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func New(pkg string) logrus.FieldLogger {
	return NewWithPrefix(pkg, pkg)
}

func NewWithPrefix(pkg string, prefix string) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(parseLevel(pkg))

	formatter := &prefixFormatter{}
	formatter.prefix = fmt.Sprintf("[%s] ", prefix)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = time.StampMicro
	logger.Formatter = formatter

	return logger
}

func parseLevel(pkg string) logrus.Level {
	lvl, ok := os.LookupEnv("LOG_" + pkg)
	if !ok {
		lvl, ok = os.LookupEnv("LOG")
	}
	if len(lvl) == 0 {
		lvl = "I"
	}

	switch lvl[0] {
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
