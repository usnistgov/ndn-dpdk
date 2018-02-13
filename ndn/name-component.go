package ndn

import (
	"bytes"
	"fmt"
	"io"
)

// A name component.
type NameComponent TlvBytes

// Test if the component is valid.
func (comp NameComponent) IsValid() bool {
	tlvType, tail := TlvBytes(comp).DecodeVarNum()
	if tail == nil || tlvType < 1 || tlvType > 32767 {
		return false
	}
	length, tail := tail.DecodeVarNum()
	if tail == nil || (TlvType(tlvType) == TT_ImplicitSha256DigestComponent && length != 32) {
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

// Print as URI.
// Implements io.WriterTo.
func (comp NameComponent) WriteTo(w io.Writer) (n int64, e error) {
	switch comp.GetType() {
	case TT_ImplicitSha256DigestComponent:
		if n2, e := fmt.Fprint(w, "sha256digest="); e != nil {
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
		if n2, e := fmt.Fprintf(w, "%v=", comp.GetType()); e != nil {
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
