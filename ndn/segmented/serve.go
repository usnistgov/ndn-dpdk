package segmented

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func makeSegmentNameComponent(seg uint64) ndn.NameComponent {
	return ndn.NameComponentFrom(an.TtSegmentNameComponent, tlv.NNI(seg))
}

func extractSegment(name ndn.Name, prefixLen int) (segment uint64, ok bool) {
	if len(name) != prefixLen+1 {
		return 0, false
	}

	comp := name.Get(-1)
	if comp.Type != an.TtSegmentNameComponent {
		return 0, false
	}

	var value tlv.NNI
	if e := value.UnmarshalBinary(comp.Value); e != nil {
		return 0, false
	}
	return uint64(value), true
}

// ServeOptions contains options for Serve function.
type ServeOptions struct {
	// ProducerOptions includes setting prefix, L3 forwarder, signer, etc.
	// Handler will be overwritten.
	endpoint.ProducerOptions

	// ContentType is Data packet ContentType.
	// Default is an.ContentBlob.
	ContentType ndn.ContentType

	// Freshness is Data packet FreshnessPeriod.
	// Default is zero.
	Freshness time.Duration

	// ChunkSize is Data payload length.
	// Default is 4096.
	ChunkSize int
}

func (opts *ServeOptions) applyDefaults() {
	opts.Handler = nil
	if opts.ChunkSize <= 0 {
		opts.ChunkSize = 4096
	}
}

// Serve publishes a segmented object.
func Serve(ctx context.Context, source io.ReaderAt, opts ServeOptions) (endpoint.Producer, error) {
	opts.applyDefaults()
	prefixLen := len(opts.Prefix)

	opts.Handler = func(ctx context.Context, interest ndn.Interest) (data ndn.Data, e error) {
		seg, ok := extractSegment(interest.Name, prefixLen)
		if !ok {
			return data, errors.New("segment component not found")
		}

		data.Name = interest.Name
		data.ContentType = opts.ContentType
		data.Freshness = opts.Freshness

		payload := make([]byte, opts.ChunkSize+1)
		n, e := source.ReadAt(payload, int64(seg)*int64(opts.ChunkSize))
		switch n {
		case 0:
			return data, e
		case opts.ChunkSize + 1:
			data.Content = payload[:opts.ChunkSize]
		default:
			data.Content = payload[:n]
			data.FinalBlock = data.Name[prefixLen]
		}
		return data, nil
	}

	return endpoint.Produce(ctx, opts.ProducerOptions)
}
