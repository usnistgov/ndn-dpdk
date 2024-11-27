// Package rdr implements Realtime Data Retrieval (RDR) protocol.
// https://redmine.named-data.net/projects/ndn-tlv/wiki/RDR
package rdr

import (
	"context"
	"encoding"
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// KeywordMetadata is the 32=metadata component.
var KeywordMetadata = ndn.MakeNameComponent(an.TtKeywordNameComponent, []byte("metadata"))

// MakeDiscoveryInterest creates an RDR discovery Interest.
// KeywordMetadata is appended automatically if it does not exist.
func MakeDiscoveryInterest(prefix ndn.Name) ndn.Interest {
	if !prefix[len(prefix)-1].Equal(KeywordMetadata) {
		prefix = prefix.Append(KeywordMetadata)
	}
	return ndn.Interest{
		Name:        prefix,
		CanBePrefix: true,
		MustBeFresh: true,
	}
}

// IsDiscoveryInterest determines whether an Interest is an RDR discovery Interest.
func IsDiscoveryInterest(interest ndn.Interest) bool {
	return len(interest.Name) > 1 && interest.Name[len(interest.Name)-1].Equal(KeywordMetadata) &&
		interest.CanBePrefix && interest.MustBeFresh
}

// RetrieveMetadata retrieves RDR metadata.
//
//	m: either *Metadata or its derived type.
func RetrieveMetadata(ctx context.Context, m encoding.BinaryUnmarshaler, name ndn.Name, opts endpoint.ConsumerOptions) error {
	interest := MakeDiscoveryInterest(name)
	data, e := endpoint.Consume(ctx, interest, opts)
	if e != nil {
		return e
	}
	if data.ContentType != an.ContentBlob {
		return ndn.ErrContentType
	}
	return m.UnmarshalBinary(data.Content)
}

// Metadata contains RDR metadata packet content.
type Metadata struct {
	Name ndn.Name
}

var (
	_ encoding.BinaryMarshaler   = Metadata{}
	_ encoding.BinaryUnmarshaler = (*Metadata)(nil)
)

// MarshalBinary encodes to TLV-VALUE.
func (m Metadata) MarshalBinary() (value []byte, e error) {
	return m.Encode()
}

// Encode encodes to TLV-VALUE with extensions.
func (m Metadata) Encode(extensions ...tlv.Fielder) (value []byte, e error) {
	return tlv.EncodeFrom(append([]tlv.Fielder{m.Name}, extensions...)...)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (m *Metadata) UnmarshalBinary(value []byte) error {
	return m.Decode(value, nil)
}

// Decode decodes from TLV-VALUE with extensions.
func (m *Metadata) Decode(value []byte, extensions MetadataDecoderMap) error {
	*m = Metadata{}
	d := tlv.DecodingBuffer(value)
	hasName := false
	for de := range d.IterElements() {
		var f MetadataFieldDecoder
		if de.Type == an.TtName && !hasName {
			f = m.decodeName
			hasName = true
		} else if f = extensions[de.Type]; f != nil {
			// use extension decoder for TLV-TYPE
		} else if f = extensions[0]; f != nil {
			// use general extension decoder
		} else {
			// ignore unknown field
			continue
		}

		if e := f(de); e != nil {
			return fmt.Errorf("TLV-TYPE 0x%02x: %w", de.Type, e)
		}
	}
	if !hasName {
		return errors.New("missing Name in RDR metadata")
	}
	return d.ErrUnlessEOF()
}

func (m *Metadata) decodeName(de tlv.DecodingElement) error {
	return de.UnmarshalValue(&m.Name)
}

// MetadataFieldDecoder is a callback function to decode a Metadata extension TLV.
type MetadataFieldDecoder func(de tlv.DecodingElement) error

// MetadataDecoderMap is a set of MetadataFieldDecoders where each key is a TLV-TYPE.
type MetadataDecoderMap map[uint32]MetadataFieldDecoder
