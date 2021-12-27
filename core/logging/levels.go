package logging

import (
	"os"

	"go.uber.org/zap"
)

// PkgLevel represents log level of a package.
type PkgLevel struct {
	pkg string
	lvl byte
	al  zap.AtomicLevel
	cb  func()
}

// Package returns package name.
func (pl PkgLevel) Package() string {
	return pl.pkg
}

// Level returns log level.
func (pl PkgLevel) Level() byte {
	return pl.lvl
}

// SetCallback sets a callback for level changing.
func (pl *PkgLevel) SetCallback(cb func()) {
	pl.cb = cb
}

// SetLevel assigns log level.
func (pl *PkgLevel) SetLevel(input string) {
	defer pl.cb()

	if len(input) == 0 {
		pl.lvl = 'I'
		pl.al.SetLevel(zap.InfoLevel)
		return
	}

	switch input[0] {
	case 'V', 'D':
		pl.al.SetLevel(zap.DebugLevel)
	case 'I':
		pl.al.SetLevel(zap.InfoLevel)
	case 'W':
		pl.al.SetLevel(zap.WarnLevel)
	case 'E':
		pl.al.SetLevel(zap.ErrorLevel)
	case 'F', 'N':
		pl.al.SetLevel(zap.DPanicLevel)
	default:
		pl.lvl = 'I'
		pl.al.SetLevel(zap.InfoLevel)
		return
	}
	pl.lvl = input[0]
}

var pkgLevels = map[string]*PkgLevel{}

// ListLevels returns all package levels.
func ListLevels() (list []PkgLevel) {
	for _, pl := range pkgLevels {
		list = append(list, *pl)
	}
	return list
}

// FindLevel returns package log level object.
func FindLevel(pkg string) (pl *PkgLevel) {
	return pkgLevels[pkg]
}

// GetLevel finds or creates package log level object.
func GetLevel(pkg string) (pl *PkgLevel) {
	pl = pkgLevels[pkg]
	if pl == nil {
		pl = &PkgLevel{
			pkg: pkg,
			al:  zap.NewAtomicLevel(),
			cb:  func() {},
		}
		pl.SetLevel(envLevel(pkg))
		pkgLevels[pkg] = pl
	}
	return pl
}

func envLevel(pkg string) string {
	v, ok := os.LookupEnv("NDNDPDK_LOG_" + pkg)
	if !ok {
		v = os.Getenv("NDNDPDK_LOG")
	}
	return v
}
