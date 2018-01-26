package dpdktestenv

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Convenient function to create testify assert and require objects.
func MakeAR(t require.TestingT) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}
