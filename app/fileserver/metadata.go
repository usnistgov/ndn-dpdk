package fileserver

import (
	"encoding"
	"math"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// KeywordMetadata is the 32=metadata component.
var KeywordMetadata = ndn.MakeNameComponent(an.TtKeywordNameComponent, []byte("metadata"))

// Assigned numbers.
const (
	TtSegmentSize = 0xF500
	TtSize        = 0xF502
	TtMode        = 0xF504
	TtAtime       = 0xF506
	TtBtime       = 0xF508
	TtCtime       = 0xF50A
	TtMtime       = 0xF50C

	_ = "enumgen::TtFileServer:Tt"
)

// Metadata represents content of RDR metadata packet from the file server.
//
// Protocol definition comes from ndn6-file-server:
// https://github.com/yoursunny/ndn6-tools/blob/main/file-server.md
type Metadata struct {
	Versioned   ndn.Name
	FinalBlock  ndn.NameComponent
	SegmentSize int
	Size        int64
	Mode        uint16
	Atime       time.Time
	Btime       time.Time
	Ctime       time.Time
	Mtime       time.Time
}

var _ encoding.BinaryUnmarshaler = (*Metadata)(nil)

// UnmarshalBinary decodes from TLV-VALUE.
func (m *Metadata) UnmarshalBinary(value []byte) (e error) {
	*m = Metadata{}
	d := tlv.DecodingBuffer(value)
	for _, de := range d.Elements() {
		switch de.Type {
		case an.TtName:
			if e = de.UnmarshalValue(&m.Versioned); e != nil {
				return e
			}
		case an.TtFinalBlock:
			d1 := tlv.DecodingBuffer(de.Value)
			if e = d1.Decode(&m.FinalBlock); e != nil {
				return e
			}
			if !m.FinalBlock.Valid() {
				return ndn.ErrComponentType
			}
			if e = d1.ErrUnlessEOF(); e != nil {
				return e
			}
		case TtSegmentSize:
			if m.SegmentSize = int(de.UnmarshalNNI(math.MaxUint16, &e, tlv.ErrRange)); e != nil {
				return e
			}
		case TtSize:
			if m.Size = int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange)); e != nil {
				return e
			}
		case TtMode:
			if m.Mode = uint16(de.UnmarshalNNI(math.MaxUint16, &e, tlv.ErrRange)); e != nil {
				return e
			}
		case TtAtime:
			if m.Atime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange))); e != nil {
				return e
			}
		case TtBtime:
			if m.Btime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange))); e != nil {
				return e
			}
		case TtCtime:
			if m.Ctime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange))); e != nil {
				return e
			}
		case TtMtime:
			if m.Mtime = time.Unix(0, int64(de.UnmarshalNNI(math.MaxInt64, &e, tlv.ErrRange))); e != nil {
				return e
			}
		}
	}
	return d.ErrUnlessEOF()
}
