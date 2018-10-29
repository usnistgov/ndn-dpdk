package ndn

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unsafe"
)

// A name component.
type NameComponent TlvBytes

// Test if the component is valid.
func (comp NameComponent) IsValid() bool {
	tlvType, tail := TlvBytes(comp).DecodeVarNum()
	if tail == nil || tlvType < 1 || tlvType > 65535 {
		return false
	}
	length, tail := tail.DecodeVarNum()
	if tail == nil || (TlvType(tlvType) == TT_ImplicitSha256DigestComponent &&
		length != implicitSha256DigestComponent_Length) {
		return false
	}
	return int(length) == len(tail)
}

// Compare equality.
func (comp NameComponent) Equal(other NameComponent) bool {
	return TlvBytes(comp).Equal(TlvBytes(other))
}

// Get TLV-TYPE.
func (comp NameComponent) GetType() TlvType {
	t, _ := TlvBytes(comp).DecodeVarNum()
	return TlvType(t)
}

// Get TLV-VALUE.
func (comp NameComponent) GetValue() TlvBytes {
	_, tail := TlvBytes(comp).DecodeVarNum() // skip TLV-TYPE
	_, tail = tail.DecodeVarNum()            // skip TLV-LENGTH
	return tail
}

const (
	implicitSha256DigestComponent_UriPrefix = "sha256digest"
	implicitSha256DigestComponent_Length    = sha256.Size
)

// Print as URI.
// Implements io.WriterTo.
func (comp NameComponent) WriteTo(w io.Writer) (n int64, e error) {
	switch comp.GetType() {
	case TT_ImplicitSha256DigestComponent:
		if n2, e := fmt.Fprintf(w, "%s=", implicitSha256DigestComponent_UriPrefix); e != nil {
			return n, e
		} else {
			n += int64(n2)
		}
		for _, b := range comp.GetValue() {
			if n2, e := fmt.Fprintf(w, "%02x", b); e != nil {
				return n, e
			} else {
				n += int64(n2)
			}
		}
		return n, e
	case TT_GenericNameComponent:
	default:
		if n2, e := fmt.Fprintf(w, "%d=", comp.GetType()); e != nil {
			return n, e
		} else {
			n += int64(n2)
		}
	}

	nNonPeriods := 0
	for _, b := range comp.GetValue() {
		if ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z') || ('0' <= b && b <= '9') ||
			b == '-' || b == '.' || b == '_' || b == '~' {
			if n2, e := fmt.Fprint(w, string(b)); e != nil {
				return n, e
			} else {
				n += int64(n2)
			}
		} else {
			if n2, e := fmt.Fprintf(w, "%%%02X", b); e != nil {
				return n, e
			} else {
				n += int64(n2)
			}
		}
		if b != '.' {
			nNonPeriods++
		}
	}
	if nNonPeriods == 0 {
		if n2, e := fmt.Fprint(w, "..."); e != nil {
			return n, e
		} else {
			n += int64(n2)
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
	tlvType := TT_GenericNameComponent
	if eqPos := strings.IndexByte(uri, '='); eqPos >= 0 {
		tlvTypeStr := uri[:eqPos]
		uri = uri[eqPos+1:]
		if tlvTypeStr == implicitSha256DigestComponent_UriPrefix {
			return parseImplicitSha256DigestComponent(uri)
		}
		if tlvTypeN, e := strconv.ParseUint(tlvTypeStr, 10, 16); e != nil {
			return nil, e
		} else {
			tlvType = TlvType(tlvTypeN)
			switch tlvType {
			case TlvType(0), TT_GenericNameComponent, TT_ImplicitSha256DigestComponent:
				return nil, errors.New("bad type indicator")
			}
		}
	}

	var buf bytes.Buffer
	if strings.TrimLeft(uri, ".") == "" {
		if len(uri) < 3 {
			return nil, errors.New("less than three periods")
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

func parseImplicitSha256DigestComponent(hexStr string) (comp NameComponent, e error) {
	value, e := hex.DecodeString(hexStr)
	if e != nil {
		return nil, e
	}
	if len(value) != implicitSha256DigestComponent_Length {
		return nil, errors.New("invalid TLV-LENGTH in ImplicitSha256DigestComponent")
	}
	return NameComponent(EncodeTlv(TT_ImplicitSha256DigestComponent, TlvBytes(value))), nil
}

// Create a name component whose TLV-VALUE is a big endian number.
func MakeNameComponentFromNumber(tlvType TlvType, v interface{}) NameComponent {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, v)
	return NameComponent(EncodeTlv(tlvType, buf.Bytes()))
}

// Join name components as TlvBytes.
func JoinNameComponents(comps []NameComponent) TlvBytes {
	return TlvBytes(bytes.Join(*(*[][]byte)(unsafe.Pointer(&comps)), nil))
}
