// Package segmented publishes and retrieves segmented objects.
package segmented

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"io"
	"math"
	"time"

	mathpkg "github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// FetchOptions contains options for Fetch function.
type FetchOptions struct {
	// Fw specifies the L3 Forwarder.
	// Default is the default Forwarder.
	Fw l3.Forwarder

	// SegmentBegin is the first segment number.
	// Default is zero.
	SegmentBegin uint64

	// SegmentEnd is the last segment number plus one.
	// Default is math.MaxUint64.
	//
	// Data FinalBlock field is always respected.
	SegmentEnd uint64

	// RetxLimit is the maximum number of retransmissions, excluding initial Interest.
	// Default is no retransmission.
	RetxLimit int

	// MaxCwnd is the maximum effective congestion window.
	// Default is no limitation.
	MaxCwnd int
}

func (opts *FetchOptions) applyDefaults() {
	if opts.SegmentEnd == 0 {
		opts.SegmentEnd = math.MaxUint64
	}
	if opts.MaxCwnd == 0 {
		opts.MaxCwnd = math.MaxInt32
	}
}

// FetchResult contains result of Fetch function.
//
// Fetching is lazy, and it starts when an output format is accessed.
// You may only access one output format on this instance, and it can be accessed only once.
type FetchResult interface {
	// Unordered emits Data packets as they arrive, not sorted in segment number order.
	Unordered(ctx context.Context, unordered chan<- *ndn.Data) error
	// Ordered emits Data packets in segment number order.
	Ordered(ctx context.Context, ordered chan<- *ndn.Data) error
	// Chunks emits Data packet payload in segment number order.
	Chunks(ctx context.Context, chunks chan<- []byte) error
	// Pipe writes the payload to the Writer.
	Pipe(ctx context.Context, w io.Writer) error
	// Packet returns a slice of Data packets.
	Packets(ctx context.Context) ([]*ndn.Data, error)
	// Payload returns reassembled payload.
	Payload(ctx context.Context) ([]byte, error)

	// Count returns the number of segments retrieved so far.
	Count() int
	// EstimatedTotal returns the estimated number of total segment.
	// Returns -1 if unknown.
	EstimatedTotal() int
}

func Fetch(name ndn.Name, opts FetchOptions) FetchResult {
	opts.applyDefaults()
	return &fetcher{
		FetchOptions: opts,
		prefix:       name,
		finalBlock:   math.MaxUint64,
	}
}

type fetcher struct {
	FetchOptions
	prefix     ndn.Name
	count      int
	finalBlock uint64
}

func (f *fetcher) makeInterest(seg uint64) ndn.Interest {
	name := append(ndn.Name{}, f.prefix...)
	name = append(name, ndn.NameComponentFrom(an.TtSegmentNameComponent, tlv.NNI(seg)))
	return ndn.MakeInterest(name)
}

func (f *fetcher) extractSeg(data *ndn.Data) (segment uint64, ok bool) {
	if len(data.Name) != len(f.prefix)+1 {
		return 0, false
	}

	comp := data.Name.Get(-1)
	if comp.Type != an.TtSegmentNameComponent {
		return 0, false
	}

	var value tlv.NNI
	if e := value.UnmarshalBinary(comp.Value); e != nil {
		return 0, false
	}
	return uint64(value), true
}

func (f *fetcher) Unordered(ctx context.Context, unordered chan<- *ndn.Data) error {
	defer close(unordered)
	face, e := endpoint.NewLFace(f.Fw)
	if e != nil {
		return e
	}
	defer face.Close()

	rtte := newRttEstimator()
	ca := newCubic()
	pendings := make(map[uint64]*fetchSeg)
	retxQ := list.New()
	ticker := time.NewTicker(time.Millisecond)
	segNext, segLast := f.SegmentBegin, f.SegmentEnd-1
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			// unblock for periodical tasks

		case l3pkt := <-face.Rx():
			pkt := l3pkt.ToPacket()
			if pkt.Data == nil {
				break
			}
			now := time.Now()

			seg, ok := f.extractSeg(pkt.Data)
			if !ok || !f.prefix.IsPrefixOf(pkt.Data.Name) {
				break
			}
			fs, ok := pendings[seg]
			if !ok {
				break
			}
			if pkt.Data.FinalBlock.Type == an.TtSegmentNameComponent {
				var finalSeg tlv.NNI
				if e := finalSeg.UnmarshalBinary(pkt.Data.FinalBlock.Value); e == nil {
					f.finalBlock = uint64(finalSeg + 1)
				}
			}

			rtt := now.Sub(fs.TxTime)
			if fs.NRetx == 0 {
				rtte.Push(rtt, len(pendings))
			}
			if pkt.Lp.CongMark != 0 {
				ca.Decrease(now)
			} else {
				ca.Increase(now, rtt)
			}

			if pkt.Data.IsFinalBlock() {
				segLast = seg
			}
			f.count++
			unordered <- pkt.Data

			if fs.RetxElement != nil {
				retxQ.Remove(fs.RetxElement)
				fs.RetxElement = nil
			}
			delete(pendings, seg)
		}

		now := time.Now()
		for seg, fs := range pendings {
			if fs.RetxElement == nil && fs.RtoExpiry.Before(now) {
				if fs.NRetx >= f.RetxLimit {
					return fmt.Errorf("exceed retx limit on segment %d", seg)
				}
				rtte.Backoff()
				ca.Decrease(fs.RtoExpiry)
				fs.RetxElement = retxQ.PushBack(seg)
			}
		}

		switch {
		case len(pendings)-retxQ.Len() >= mathpkg.Min(ca.Cwnd(), f.MaxCwnd):
			// congestion window full

		case retxQ.Len() > 0:
			seg := retxQ.Remove(retxQ.Front()).(uint64)
			fs := pendings[seg]
			fs.RetxElement = nil

			fs.setTimeNow(rtte.Rto())
			fs.NRetx++
			face.Tx() <- f.makeInterest(seg).ToPacket()

		case segNext <= segLast:
			seg := segNext
			segNext++

			fs := &fetchSeg{}
			fs.setTimeNow(rtte.Rto())
			pendings[seg] = fs
			face.Tx() <- f.makeInterest(seg).ToPacket()

		case len(pendings) == 0:
			return nil
		}
	}
}

func (f *fetcher) Ordered(ctx context.Context, ordered chan<- *ndn.Data) error {
	defer close(ordered)
	unordered := make(chan *ndn.Data)
	done := make(chan error)
	go func() { done <- f.Unordered(ctx, unordered) }()

	next := f.SegmentBegin
	buffer := make(map[uint64]*ndn.Data)
	for data := range unordered {
		seg, ok := f.extractSeg(data)
		switch {
		case !ok, seg < next:
			continue
		case seg == next:
			ordered <- data
			next++
			for {
				data, ok = buffer[next]
				if !ok {
					break
				}
				delete(buffer, next)
				ordered <- data
				next++
			}
		case seg > next:
			buffer[seg] = data
		}
	}

	if e := <-done; e != nil {
		return e
	}
	if n := len(buffer); n > 0 {
		return fmt.Errorf("%d segments are not reassembled", len(buffer))
	}
	return nil
}

func (f *fetcher) Chunks(ctx context.Context, chunks chan<- []byte) error {
	defer close(chunks)
	ordered := make(chan *ndn.Data)
	done := make(chan error)
	go func() { done <- f.Ordered(ctx, ordered) }()
	for data := range ordered {
		chunks <- data.Content
	}
	return <-done
}

func (f *fetcher) Pipe(ctx context.Context, w io.Writer) error {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	chunks := make(chan []byte)
	done := make(chan error)
	go func() { done <- f.Chunks(ctx, chunks) }()
	for chunk := range chunks {
		if _, e := w.Write(chunk); e != nil {
			return e
		}
	}

	cancel()
	return <-done
}

func (f *fetcher) Packets(ctx context.Context) (packets []*ndn.Data, e error) {
	ordered := make(chan *ndn.Data)
	done := make(chan error)
	go func() { done <- f.Ordered(ctx, ordered) }()

	for packet := range ordered {
		packets = append(packets, packet)
	}
	return packets, <-done
}

func (f *fetcher) Payload(ctx context.Context) ([]byte, error) {
	ordered := make(chan []byte)
	done := make(chan error)
	go func() { done <- f.Chunks(ctx, ordered) }()

	chunks := make([][]byte, 0)
	for chunk := range ordered {
		chunks = append(chunks, chunk)
	}
	if e := <-done; e != nil {
		return nil, e
	}
	return bytes.Join(chunks, nil), nil
}

func (f *fetcher) Count() int {
	return f.count
}

func (f *fetcher) EstimatedTotal() int {
	segLast := mathpkg.MinUint64(f.SegmentEnd, f.finalBlock)
	if segLast == math.MaxUint64 {
		return -1
	}
	return int(segLast - f.SegmentBegin + 1)
}
