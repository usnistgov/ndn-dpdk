package testenv

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/stretchr/testify/assert"
)

// BytesFromHex converts a hexadecimal string to a byte slice.
// The octets must be written as upper case.
// All characters other than [0-9A-F] are considered comments and stripped.
func BytesFromHex(input string) []byte {
	s := strings.Map(func(ch rune) rune {
		if strings.ContainsRune("0123456789ABCDEF", ch) {
			return ch
		}
		return -1
	}, input)
	decoded, e := hex.DecodeString(s)
	if e != nil {
		panic(fmt.Errorf("hex.DecodeString error %w", e))
	}
	return decoded
}

// BytesEqual asserts that actual bytes equals expected bytes.
// It considers nil slice and zero-length slice to be the same.
func BytesEqual(a *assert.Assertions, expected, actual []byte, msgAndArgs ...interface{}) bool {
	if len(expected) == 0 && len(actual) == 0 {
		return true
	}
	return a.Equal(expected, actual, msgAndArgs...)
}
