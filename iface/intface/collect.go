package intface

import (
	"sync"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Collector accumulates packets received by ndn.Face.
type Collector struct {
	received []*ndn.Packet
	lock     sync.RWMutex
}

// Collect starts collecting packets received by face.
func Collect(f *IntFace) *Collector {
	var collector Collector
	go collector.run(f.A)
	return &collector
}

func (c *Collector) run(face ndn.L3Face) {
	for packet := range face.Rx() {
		c.lock.Lock()
		c.received = append(c.received, packet)
		c.lock.Unlock()
	}
}

// Clear deletes collected packets.
func (c *Collector) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.received = nil
}

// Peek provides access to the slice of collected packets.
func (c *Collector) Peek(f func(received []*ndn.Packet)) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	f(c.received)
}

// Count returns number of collected packets.
func (c *Collector) Count() (count int) {
	c.Peek(func(received []*ndn.Packet) { count = len(received) })
	return count
}

// Get returns i-th collected packets.
// If negative, count from the end.
// If out-of-range, return nil.
func (c *Collector) Get(i int) (packet *ndn.Packet) {
	c.Peek(func(received []*ndn.Packet) {
		if i < 0 {
			i += len(received)
		}
		if i >= 0 && i < len(received) {
			packet = received[i]
		}
	})
	return packet
}
