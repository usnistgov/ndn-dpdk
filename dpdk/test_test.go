package dpdk

// This file contains test setup procedure and common test helper functions.

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var testEal *Eal

func TestMain(m *testing.M) {
	eal, e := NewEal([]string{"testprog", "-n1"})
	if e != nil || eal == nil {
		panic(fmt.Sprintf("NewEal error %v", e))
	}
	os.Exit(m.Run())
}

func makeAR(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}
