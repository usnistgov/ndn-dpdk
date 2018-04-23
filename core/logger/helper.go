package logger

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

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

func AddressOf(ptr interface{}) string {
	return fmt.Sprintf("%p", ptr)
}
