package fwdptest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func TestFastroute(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3, collect4 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3), intface.Collect(face4)
	fixture.SetFibEntry("/A/B", "fastroute", face1.ID, face2.ID, face3.ID)

	// multicast first Interest
	face4.Tx <- ndn.MakeInterest("/A/B/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	// face3 replies Data
	face3.Tx <- ndn.MakeData(collect3.Get(-1).Interest)
	fixture.StepDelay()

	// unicast to face3
	face4.Tx <- ndn.MakeInterest("/A/B/2")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(2, collect3.Count())

	// unicast to face3
	face4.Tx <- ndn.MakeInterest("/A/B/3")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(3, collect3.Count())

	// face3 fails
	face3.SetDown(true)

	// multicast next Interest because face3 failed
	face4.Tx <- ndn.MakeInterest("/A/B/4")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(2, collect2.Count())
	assert.Equal(3, collect3.Count()) // no Interest to face3 because it's DOWN

	// face1 replies Data
	face1.Tx <- ndn.MakeData(collect1.Get(-1).Interest)
	fixture.StepDelay()

	// unicast to face1
	face4.Tx <- ndn.MakeInterest("/A/B/5", ndn.NonceFromUint(0x422e9f49))
	fixture.StepDelay()
	assert.Equal(3, collect1.Count())
	assert.Equal(2, collect2.Count())
	assert.Equal(3, collect3.Count())

	// face1 replies Nack~NoRoute, retry on other faces
	face1.Tx <- ndn.MakeNack(collect1.Get(-1).Interest, an.NackNoRoute)
	fixture.StepDelay()
	assert.Equal(3, collect1.Count())
	assert.Equal(3, collect2.Count())
	assert.Equal(3, collect3.Count()) // no Interest to face3 because it's DOWN

	// face2 replies Nack~NoRoute as well, return Nack to downstream
	collect4.Clear()
	face2.Tx <- ndn.MakeNack(collect2.Get(-1).Interest, an.NackNoRoute)
	fixture.StepDelay()
	assert.Equal(1, collect4.Count())
	assert.NotNil(collect4.Get(-1).Nack)

	// multicast next Interest because faces Nacked
	face4.Tx <- ndn.MakeInterest("/A/B/6")
	fixture.StepDelay()
	assert.Equal(4, collect1.Count())
	assert.Equal(4, collect2.Count())
	assert.Equal(3, collect3.Count()) // no Interest to face3 because it's DOWN
}

func TestFastrouteProbe(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	fixture.SetFibEntry("/F", "fastroute", face1.ID, face2.ID, face3.ID)

	ctx, cancel := context.WithCancel(context.TODO())
	var wg sync.WaitGroup
	defer func() {
		cancel()
		wg.Wait()
	}()
	startConsumer := func() { // 500 Interests per second
		wg.Add(1)
		go func() {
			defer wg.Done()
			tick := time.NewTicker(2 * time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case t := <-tick.C:
					face4.Tx <- ndn.MakeInterest(fmt.Sprintf("/F/F/%d", t.UnixNano()))
				case <-face4.Rx:
				}
			}
		}()
	}
	startProducer := func(face *intface.IntFace) (cnt *int, delay *time.Duration) {
		type ProbeRecord struct {
			Timer <-chan time.Time
			Data  ndn.Data
		}
		cnt = new(int)
		delay = new(time.Duration)
		queue := make(chan ProbeRecord, 65536)
		wg.Add(2)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case record := <-queue:
					<-record.Timer
					face.Tx <- record.Data
				}
			}
		}()
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case pkt := <-face.Rx:
					if pkt.Interest != nil {
						*cnt++
						queue <- ProbeRecord{
							Timer: time.After(*delay),
							Data:  ndn.MakeData(pkt.Interest),
						}
					}
				}
			}
		}()
		return
	}
	cnt1, delay1 := startProducer(face1)
	cnt2, delay2 := startProducer(face2)
	cnt3, delay3 := startProducer(face3)
	startConsumer()

	// face2 is fastest
	*delay1, *delay2, *delay3 = 40*time.Millisecond, 1*time.Millisecond, 40*time.Millisecond
	time.Sleep(1 * time.Second)
	*cnt1, *cnt2, *cnt3 = 0, 0, 0
	time.Sleep(2 * time.Second)
	assert.Greater(*cnt2/4, *cnt1)
	assert.Greater(*cnt2/4, *cnt3)

	// face1 is fastest
	*delay1, *delay2, *delay3 = 1*time.Millisecond, 40*time.Millisecond, 40*time.Millisecond
	// consumer sends 500 I/s, probe occurs every 1024 Interests, so there must be a probe within 8 seconds
	time.Sleep(8 * time.Second)
	*cnt1, *cnt2, *cnt3 = 0, 0, 0
	time.Sleep(2 * time.Second)
	assert.Greater(*cnt1/4, *cnt2)
	assert.Greater(*cnt1/4, *cnt3)
}

func TestSequential(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/A", "sequential", face1.ID, face2.ID, face3.ID)

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(0, collect2.Count())
	assert.Equal(0, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(0, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face2.SetDown(true)

	face4.Tx <- ndn.MakeInterest("/A/1")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(2, collect3.Count())
}

func TestRoundrobin(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/A/B", "roundrobin", face1.ID, face2.ID, face3.ID)

	face4.Tx <- ndn.MakeInterest("/A/B/0")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(0, collect2.Count())
	assert.Equal(0, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/B/1")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(0, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/B/2")
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/B/3")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face2.SetDown(true)

	face4.Tx <- ndn.MakeInterest("/A/B/4")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face4.Tx <- ndn.MakeInterest("/A/B/5")
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(2, collect3.Count())
}

func TestWeighted(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	w1, w2, w3 := 6.0, 3.0, 1.0
	fixture.SetFibEntryParams("/P", "weighted", map[string]interface{}{"weights": []float64{w1, w2, w3}},
		face1.ID, face2.ID, face3.ID)

	for i := 0; i < 200; i++ {
		face1.Tx <- ndn.MakeInterest(fmt.Sprintf("/P/1/%d", i))
		face2.Tx <- ndn.MakeInterest(fmt.Sprintf("/P/2/%d", i))
		face4.Tx <- ndn.MakeInterest(fmt.Sprintf("/P/4/%d", i))
		time.Sleep(time.Millisecond)
	}
	fixture.StepDelay()

	countInterests := func(c *intface.Collector, n1, n2, n4 *int) {
		c.Peek(func(received []*ndn.Packet) {
			for _, p := range received {
				if assert.NotNil(p.Interest) && assert.Len(p.Interest.Name, 3) && assert.Len(p.Interest.Name[1].Value, 1) {
					switch p.Interest.Name[1].Value[0] {
					case '1':
						*n1++
					case '2':
						*n2++
					case '4':
						*n4++
					default:
						assert.Fail("unexpected Interest name", p.Interest.Name.String())
					}
				}
			}
		})
	}

	var n11, n12, n13, n21, n22, n23, n41, n42, n43 int
	countInterests(collect1, &n11, &n21, &n41)
	countInterests(collect2, &n12, &n22, &n42)
	countInterests(collect3, &n13, &n23, &n43)
	assert.Zero(n11)
	assert.InDelta(w2/(w2+w3), float64(n12)/float64(n12+n13), 0.1)
	assert.Zero(n22)
	assert.InDelta(w1/(w1+w3), float64(n21)/float64(n21+n23), 0.1)
	assert.InDelta(w1/(w1+w2+w3), float64(n41)/float64(n41+n42+n43), 0.1)
	assert.InDelta(w2/(w1+w2+w3), float64(n42)/float64(n41+n42+n43), 0.1)
}
