package endpoint

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"go4.org/must"
)

// Error conditions.
var (
	//lint:ignore ST1005 'Handler' is a field name
	ErrNoHandler = errors.New("Handler is missing")
)

type producerNackError uint8

func (producerNackError) Error() string {
	return "Nack"
}

// ReplyNack causes the producer to return a Nack packet.
func ReplyNack(reason uint8) error {
	return producerNackError(reason)
}

// ProducerHandler is a producer handler function.
//  - If it returns an error created with ReplyNack(), a Nack is sent in reply to the Interest.
//  - If it returns a Data that satisfies the Interest, the Data is sent in reply to the Interest.
//  - Otherwise, nothing is sent.
type ProducerHandler func(ctx context.Context, interest ndn.Interest) (ndn.Data, error)

// ProducerOptions contains arguments to Produce function.
type ProducerOptions struct {
	// Prefix is the name prefix of the producer.
	Prefix ndn.Name

	// NoAdvertise disables prefix announcement.
	// Default is announcing the prefix.
	NoAdvertise bool

	// Handler is a function to handle Interests under the prefix.
	// This may be invoked concurrently.
	Handler ProducerHandler

	// Fw specifies the L3 Forwarder.
	// Default is the default Forwarder.
	Fw l3.Forwarder

	// DataSigner automatically signs Data packets unless already signed.
	// Default is keeping the Null signature.
	DataSigner ndn.Signer
}

func (opts *ProducerOptions) applyDefaults() {
	if opts.Fw == nil {
		opts.Fw = l3.GetDefaultForwarder()
	}
}

// Produce starts a producer.
func Produce(ctx context.Context, opts ProducerOptions) (Producer, error) {
	opts.applyDefaults()
	if opts.Handler == nil {
		return nil, ErrNoHandler
	}

	face, e := newLFace(opts.Fw)
	if e != nil {
		return nil, e
	}
	face.fwFace.AddRoute(opts.Prefix)
	if !opts.NoAdvertise {
		face.fwFace.AddAnnouncement(opts.Prefix)
	}

	ctx1, cancel := context.WithCancel(ctx)
	p := &producer{
		ProducerOptions: opts,
		face:            face,
		close:           cancel,
	}
	go p.loop(ctx1)
	return p, nil
}

// Producer represents a running producer.
type Producer interface {
	io.Closer
}

type producer struct {
	ProducerOptions
	face  *lFace
	close context.CancelFunc
}

func (p *producer) Close() error {
	p.close()
	return nil
}

func (p *producer) loop(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	defer func() {
		wg.Wait()
		must.Close(p.face)
		p.close()
	}()

L:
	for {
		select {
		case <-ctx.Done():
			wg.Done()
			return
		case l3pkt := <-p.face.fw2ep:
			pkt := l3pkt.ToPacket()
			if pkt.Interest == nil {
				continue L
			}
			wg.Add(1)
			go p.handleInterest(ctx, &wg, pkt)
		}
	}
}

func (p *producer) handleInterest(ctx context.Context, wg *sync.WaitGroup, pkt *ndn.Packet) {
	defer wg.Done()

	interest := pkt.Interest
	if !p.Prefix.IsPrefixOf(interest.Name) {
		return
	}

	ctx1, cancel := context.WithTimeout(ctx, interest.ApplyDefaultLifetime())
	defer cancel()
	data, e := p.Handler(ctx1, *interest)

	var reply *ndn.Packet
	if e != nil {
		if nackError, ok := e.(producerNackError); ok {
			nack := ndn.MakeNack(interest, uint8(nackError))
			reply = nack.ToPacket()
		}
	} else if data.CanSatisfy(*interest) {
		if (data.SigInfo == nil || data.SigInfo.Type == an.SigNull) && p.DataSigner != nil {
			if e := p.DataSigner.Sign(&data); e != nil {
				return
			}
		}
		reply = &ndn.Packet{
			Lp:   pkt.Lp,
			Data: &data,
		}
	}

	if reply == nil {
		return
	}
	select {
	case <-ctx.Done():
	case p.face.ep2fw <- reply:
	}
}
