package endpoint

import (
	"context"
	"errors"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Error conditions.
var (
	ErrExpire = errors.New("Interest expired")
)

// ConsumerOptions contains arguments to Consume function.
type ConsumerOptions struct {
	// Fw specifies the L3 Forwarder.
	// Default is the default Forwarder.
	Fw l3.Forwarder

	// Retx specifies retransmission policy.
	// Default is disabling retransmission.
	Retx RetxPolicy

	// Verifier specifies a Data verifier.
	// Default is no verification.
	Verifier ndn.Verifier
}

func (opts *ConsumerOptions) applyDefaults() {
	if opts.Fw == nil {
		opts.Fw = l3.GetDefaultForwarder()
	}
	if opts.Retx == nil {
		opts.Retx = noRetx{}
	}
	if opts.Verifier == nil {
		opts.Verifier = ndn.NopVerifier
	}
}

// Consume retrieves a single piece of Data.
func Consume(ctx context.Context, interest ndn.Interest, opts ConsumerOptions) (data *ndn.Data, e error) {
	opts.applyDefaults()
	face, e := newLFace(opts.Fw)
	if e != nil {
		return nil, e
	}
	defer face.Close()

	retxIntervals := opts.Retx.IntervalIterable(interest.ApplyDefaultLifetime())
L:
	for {
		var timer *time.Timer
		rto := retxIntervals()
		if rto > 0 {
			timer = time.NewTimer(rto)
		} else {
			timer = time.NewTimer(interest.Lifetime)
		}

		face.ep2fw <- interest.ToPacket()

		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-timer.C:
				if rto > 0 {
					continue L
				}
				return nil, ErrExpire
			case l3pkt := <-face.fw2ep:
				data = l3pkt.ToPacket().Data
				if data != nil && data.CanSatisfy(interest) {
					break L
				}
			}
		}
	}

	if e := opts.Verifier.Verify(data); e != nil {
		return nil, e
	}
	return data, nil
}
