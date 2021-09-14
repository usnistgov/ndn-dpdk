// Package rdr implements ndn6-file-server protocol.
// https://github.com/yoursunny/ndn6-tools/blob/main/file-server.md
package ndn6file

import (
	"encoding"
	"math"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/rdr"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Assigned numbers.
const (
	TtSegmentSize = 0xF500
	TtSize        = 0xF502
	TtMode        = 0xF504
	TtAtime       = 0xF506
	TtBtime       = 0xF508
	TtCtime       = 0xF50A
	TtMtime       = 0xF50C

	_ = "enumgen::TtFile:Tt"
)

const (
	sIFREG = 0x8000
	sIFDIR = 0x4000
)

// Metadata represents RDR metadata packet with file server extensions.
type Metadata struct {
	rdr.Metadata
	FinalBlock  ndn.NameComponent
	SegmentSize int
	Size        int64
	Mode        uint16
	Atime       time.Time
	Btime       time.Time
	Ctime       time.Time
	Mtime       time.Time
}

var (
	_ encoding.BinaryMarshaler   = Metadata{}
	_ encoding.BinaryUnmarshaler = (*Metadata)(nil)
)

// IsFile determines whether Mode indicates a regular file.
func (m Metadata) IsFile() bool {
	return (m.Mode & sIFREG) == sIFREG
}

// IsDir determines whether Mode indicates a directory.
func (m Metadata) IsDir() bool {
	return (m.Mode & sIFDIR) == sIFDIR
}

// MarshalBinary encodes to TLV-VALUE.
func (m Metadata) MarshalBinary() (value []byte, e error) {
	extensions := []tlv.Fielder{}
	if m.FinalBlock.Valid() {
		extensions = append(extensions, tlv.TLVFrom(an.TtFinalBlock, m.FinalBlock))
	}
	if m.SegmentSize > 0 {
		extensions = append(extensions, tlv.TLVNNI(TtSegmentSize, uint64(m.SegmentSize)))
		extensions = append(extensions, tlv.TLVNNI(TtSize, uint64(m.Size)))
	}
	extensions = append(extensions, tlv.TLVNNI(TtMode, uint64(m.Mode)))
	if !m.Atime.IsZero() {
		extensions = append(extensions, tlv.TLVNNI(TtAtime, uint64(m.Atime.UnixNano())))
	}
	if !m.Btime.IsZero() {
		extensions = append(extensions, tlv.TLVNNI(TtBtime, uint64(m.Btime.UnixNano())))
	}
	if !m.Ctime.IsZero() {
		extensions = append(extensions, tlv.TLVNNI(TtCtime, uint64(m.Ctime.UnixNano())))
	}
	if !m.Mtime.IsZero() {
		extensions = append(extensions, tlv.TLVNNI(TtMtime, uint64(m.Mtime.UnixNano())))
	}
	return m.Encode(extensions...)
}

// UnmarshalBinary decodes from TLV-VALUE.
func (m *Metadata) UnmarshalBinary(value []byte) (e error) {
	return m.Metadata.Decode(value, rdr.MetadataDecoderMap{
		an.TtFinalBlock: func(de tlv.DecodingElement) error {
			d := tlv.DecodingBuffer(de.Value)
			if e = d.Decode(&m.FinalBlock); e != nil {
				return e
			}
			if !m.FinalBlock.Valid() {
				return ndn.ErrComponentType
			}
			return d.ErrUnlessEOF()
		},
		TtSegmentSize: func(de tlv.DecodingElement) (e error) {
			m.SegmentSize = int(de.UnmarshalNNI(math.MaxUint16, &e, tlv.ErrRange))
			return e
		},
		TtSize: func(de tlv.DecodingElement) (e error) {
			m.Size = int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange))
			return e
		},
		TtMode: func(de tlv.DecodingElement) (e error) {
			m.Mode = uint16(de.UnmarshalNNI(math.MaxUint16, &e, tlv.ErrRange))
			return e
		},
		TtAtime: func(de tlv.DecodingElement) (e error) {
			m.Atime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange)))
			return e
		},
		TtBtime: func(de tlv.DecodingElement) (e error) {
			m.Btime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange)))
			return e
		},
		TtCtime: func(de tlv.DecodingElement) (e error) {
			m.Ctime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange)))
			return e
		},
		TtMtime: func(de tlv.DecodingElement) (e error) {
			m.Mtime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange)))
			return e
		},
	})
}
