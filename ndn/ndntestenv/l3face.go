package ndntestenv

import (
	"encoding/binary"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// L3FaceTester tests L3Face or Transport.
type L3FaceTester struct {
	Count            int
	LossTolerance    float64
	InterestInterval time.Duration
	CloseDelay       time.Duration
}

func (c *L3FaceTester) applyDefaults() {
	if c.Count <= 0 {
		c.Count = 1000
	}
	if c.LossTolerance <= 0.0 {
		c.LossTolerance = 0.05
	}
	if c.InterestInterval <= 0 {
		c.InterestInterval = 1 * time.Millisecond
	}
	if c.CloseDelay <= 0 {
		c.CloseDelay = 100 * time.Millisecond
	}
}

// CheckTransport tests a pair of connected Transport.
func (c *L3FaceTester) CheckTransport(t *testing.T, trA, trB ndn.Transport) {
	_, require := testenv.MakeAR(t)
	faceA, e := ndn.NewL3Face(trA)
	require.NoError(e)
	faceB, e := ndn.NewL3Face(trB)
	require.NoError(e)
	c.CheckL3Face(t, faceA, faceB)
}

// CheckL3Face tests a pair of connected L3Face.
func (c *L3FaceTester) CheckL3Face(t *testing.T, faceA, faceB ndn.L3Face) {
	c.applyDefaults()
	assert, require := testenv.MakeAR(t)

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		txB := faceB.GetTx()
		for packet := range faceB.GetRx() {
			require.NotNil(packet.Interest)
			data := ndn.MakeData(packet.Interest.Name)
			var reply ndn.Packet
			reply.Data = &data
			reply.Lp.PitToken = packet.Lp.PitToken
			txB <- &reply
		}
		wg.Done()
	}()

	nData := 0
	hasData := make([]bool, c.Count)
	go func() {
		for packet := range faceA.GetRx() {
			require.NotNil(packet.Data)
			require.Len(packet.Lp.PitToken, 8)
			token := binary.LittleEndian.Uint64(packet.Lp.PitToken)
			require.LessOrEqual(token, uint64(c.Count), "%d", token)
			assert.False(hasData[token], "%d", token)
			hasData[token] = true
			nData++
		}
		wg.Done()
	}()

	go func() {
		txA := faceA.GetTx()
		for i := 0; i < c.Count; i++ {
			interest := ndn.MakeInterest(fmt.Sprintf("/A/%d", i))
			var packet ndn.Packet
			packet.Interest = &interest
			packet.Lp.PitToken = make([]byte, 8)
			binary.LittleEndian.PutUint64(packet.Lp.PitToken, uint64(i))
			txA <- &packet
			time.Sleep(c.InterestInterval)
		}

		time.Sleep(c.CloseDelay)
		require.NoError(faceA.Close())
		require.NoError(faceB.Close())
		wg.Done()
	}()

	wg.Wait()
	assert.InEpsilon(c.Count, nData, c.LossTolerance)
}
