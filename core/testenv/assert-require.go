// Package testenv provides general test utilities.
package testenv

import (
	"math/rand"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// MakeAR creates testify assert and require objects.
func MakeAR(t require.TestingT) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}
