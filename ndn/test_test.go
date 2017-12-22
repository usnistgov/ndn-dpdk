package ndn

// This file contains test setup procedure and common test helper functions.

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(255, 0, 2000)

	os.Exit(m.Run())
}

func makeAR(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}

func packetFromHex(input string) dpdk.Packet {
	return dpdktestenv.PacketFromHex(input)
}
