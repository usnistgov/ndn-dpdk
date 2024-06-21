//go:build !cgo

package testenv

import "os"

func Exit(code int) {
	os.Exit(code)
}
