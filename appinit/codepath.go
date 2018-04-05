package appinit

import (
	"path"
	"runtime"
)

var CodePath string

func init() {
	if _, filePath, _, ok := runtime.Caller(0); ok {
		CodePath = path.Join(path.Dir(filePath), "..")
	}
}
