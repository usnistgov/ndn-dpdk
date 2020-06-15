package logger

import (
	"github.com/sirupsen/logrus"
)

// MakeFields constructs logrus.Fields from a sequence of arguments.
// Arguments must appear in key-value pairs.
func MakeFields(args ...interface{}) logrus.Fields {
	fields := make(logrus.Fields)
	key := ""
	for _, a := range args {
		if key == "" {
			key = a.(string)
		} else {
			fields[key] = a
			key = ""
		}
	}
	return fields
}
