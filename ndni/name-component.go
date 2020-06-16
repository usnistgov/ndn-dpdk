package ndni

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// A name component.
type NameComponent TlvBytes

// Test if a TLV-TYPE is valid as a component type.
func IsValidNameComponentType(tt an.TlvType) bool {
	return tt > 0 && tt <= 65535
}

// Test if the component is valid.
func (comp NameComponent) IsValid() bool {
	tlvType, tail := TlvBytes(comp).DecodeVarNum()
	length, tail := tail.DecodeVarNum()
	return tail != nil && int(length) == len(tail) && IsValidNameComponentType(an.TlvType(tlvType))
}

// Compare equality.
func (comp NameComponent) Equal(other NameComponent) bool {
	return TlvBytes(comp).Equal(TlvBytes(other))
}

// Get TLV-TYPE.
func (comp NameComponent) GetType() an.TlvType {
	t, _ := TlvBytes(comp).DecodeVarNum()
	return an.TlvType(t)
}

// Get TLV-VALUE.
func (comp NameComponent) GetValue() TlvBytes {
	_, tail := TlvBytes(comp).DecodeVarNum() // skip TLV-TYPE
	_, tail = tail.DecodeVarNum()            // skip TLV-LENGTH
	return tail
}

// Print as URI in canonical format.
// Implements io.WriterTo.
func (comp NameComponent) WriteTo(w io.Writer) (n int64, e error) {
	if c, e := fmt.Fprintf(w, "%d=", comp.GetType()); e != nil {
		return n, e
	} else {
		n += int64(c)
	}

	nNonPeriods := 0
	for _, b := range comp.GetValue() {
		var c int
		if ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') ||
			b == '-' || b == '.' || b == '_' || b == '~' {
			c, e = fmt.Fprint(w, string(b))
		} else {
			c, e = fmt.Fprintf(w, "%%%02X", b)
		}

		if e != nil {
			return n, e
		} else {
			n += int64(c)
		}

		if b != '.' {
			nNonPeriods++
		}
	}

	if nNonPeriods == 0 {
		if c, e := fmt.Fprint(w, "..."); e != nil {
			return n, e
		} else {
			n += int64(c)
		}
	}
	return n, nil
}

// Convert to URI.
func (comp NameComponent) String() string {
	var sb bytes.Buffer
	comp.WriteTo(&sb)
	return sb.String()
}

// Parse name component from URI.
func ParseNameComponent(uri string) (comp NameComponent, e error) {
	tlvType := an.TtGenericNameComponent
	if eqPos := strings.IndexByte(uri, '='); eqPos >= 0 {
		if tlvTypeN, e := strconv.ParseUint(uri[:eqPos], 10, 16); e != nil {
			return nil, e
		} else {
			tlvType = an.TlvType(tlvTypeN)
			uri = uri[eqPos+1:]
		}
	}

	var buf bytes.Buffer
	if strings.TrimLeft(uri, ".") == "" {
		if len(uri) < 3 {
			return nil, errors.New("fewer than three periods")
		}
		buf.WriteString(uri[3:])
	} else {
		for i := 0; i < len(uri); i++ {
			ch := uri[i]
			if ch == '%' && i+2 < len(uri) {
				b, e := hex.DecodeString(uri[i+1 : i+3])
				if e != nil {
					return nil, fmt.Errorf("hex error near position %d: %v", i, e)
				}
				buf.Write(b)
				i += 2
			} else {
				buf.WriteByte(ch)
			}
		}
	}

	return NameComponent(EncodeTlv(tlvType, buf.Bytes())), nil
}
