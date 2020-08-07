// Package testenv provides general test utilities.
package testenv

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MakeAR creates testify assert and require objects.
func MakeAR(t require.TestingT) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}
