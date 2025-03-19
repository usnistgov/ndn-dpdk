// Package testenv provides general test utilities.
package testenv

import (
	"math"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"
)

// MakeAR creates testify assert and require objects.
func MakeAR(t require.TestingT) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

// Between asserts that actual is between two bounds (inclusive).
//
//	lbound <= actual <= ubound
func Between[A constraints.Integer, L constraints.Integer, U constraints.Integer](
	a *assert.Assertions, lbound L, ubound U, actual A, msgAndArgs ...any,
) bool {
	return a.GreaterOrEqual(int64(actual), int64(lbound), msgAndArgs...) &&
		a.LessOrEqual(int64(actual), int64(ubound), msgAndArgs...)
}

// AtOrBelow asserts that actual is no more than expected and has bounded relative error.
//
//	expected * (1-tolerance) <= actual <= expected
func AtOrBelow[A constraints.Integer, E constraints.Integer, L constraints.Float](
	a *assert.Assertions, expected E, actual A, tolerance L, msgAndArgs ...any,
) bool {
	lbound := math.Ceil(float64(expected) * (1 - float64(tolerance)))
	return Between(a, int64(lbound), expected, actual, msgAndArgs...)
}

// AtOrAround asserts that actual is near expected and has bounded relative error.
//
//	expected * (1-lTolerance) <= actual <= expected * (1+uTolerance)
func AtOrAround[A constraints.Integer, E constraints.Integer, L constraints.Float, U constraints.Float](
	a *assert.Assertions, expected E, actual A, lTolerance L, uTolerance U, msgAndArgs ...any,
) bool {
	lbound := math.Ceil(float64(expected) * (1 - float64(lTolerance)))
	ubound := math.Floor(float64(expected) * (1 + float64(uTolerance)))
	return Between(a, int64(lbound), int64(ubound), actual, msgAndArgs...)
}
