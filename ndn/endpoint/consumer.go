package endpoint

import (
	"context"
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Error conditions.
var (
	//lint:ignore ST1005 'Interest' is a proper noun
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
	face, e := NewLFace(opts.Fw)
	if e != nil {
		return nil, e
	}
	defer face.Close()

	c := &consumer{
		face:     face,
		interest: interest,
		retxIter: opts.Retx.IntervalIterable(interest.ApplyDefaultLifetime()),
	}
	for !c.retxEnd {
		data, e = c.once(ctx)
		switch e {
		case nil:
			if e = opts.Verifier.Verify(data); e != nil {
				return nil, e
			}
			return data, nil
		case context.DeadlineExceeded:
			if e = ctx.Err(); e != nil { // parent context timeout
				return nil, e
			}
			// per-packet context timeout, proceed to retransmission
		default:
			return nil, e
		}
	}
	return nil, ErrExpire
}

type consumer struct {
	face     *LFace
	interest ndn.Interest
	retxIter RetxIterable
	retxEnd  bool
}

func (c *consumer) once(ctx context.Context) (data *ndn.Data, e error) {
	rto := c.retxIter()
	if rto == 0 {
		c.retxEnd = true
		rto = c.interest.Lifetime
	}
	ctx1, cancel1 := context.WithTimeout(ctx, rto)
	defer cancel1()

	select {
	case c.face.Tx() <- c.interest.ToPacket():
	case <-ctx1.Done():
		return nil, ctx1.Err()
	}

	for {
		select {
		case <-ctx1.Done():
			return nil, ctx1.Err()
		case l3pkt := <-c.face.Rx():
			data := l3pkt.ToPacket().Data
			if data != nil && data.CanSatisfy(c.interest) {
				return data, nil
			}
		}
	}
}
