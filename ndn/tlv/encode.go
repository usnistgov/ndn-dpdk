package tlv

import (
	"bytes"

	"math"
)

// EncodingBuffer is an encoding buffer.
// Zero value is an empty buffer.
type EncodingBuffer struct {
	b   []byte
	err error
}

// Append appends a field.
func (eb *EncodingBuffer) Append(f Field) {
	if eb.err != nil {
		return
	}
	eb.b, eb.err = f.Encode(eb.b)
}

// Output returns encoding output.
func (eb EncodingBuffer) Output() ([]byte, error) {
	return eb.b, eb.err
}

type fieldType uint8

const (
	fieldTypeEmpty fieldType = iota
	fieldTypeError
	fieldTypeBytes
	fieldTypeNNI
	fieldTypeTLVFields
	fieldTypeTLVFielders
)

// Field is an encodable field.
// Zero value encodes to nothing.
type Field struct {
	typ     fieldType
	integer uint64
	object  interface{}
}

// Encode appends to the byte slice.
func (f Field) Encode(b []byte) (o []byte, e error) {
	switch f.typ {
	case fieldTypeEmpty:
		return b, nil
	case fieldTypeError:
		return nil, f.object.(error)
	case fieldTypeBytes:
		return append(b, f.object.([]byte)...), nil
	case fieldTypeNNI:
		return NNI(f.integer).Encode(b), nil
	case fieldTypeTLVFields:
		return f.encodeTLVFields(b)
	case fieldTypeTLVFielders:
		return f.encodeTLVFielders(b)
	default:
		panic(f.typ)
	}
}

func (f Field) encodeTLVFields(b []byte) (o []byte, e error) {
	subs := f.object.([]Field)
	parts := make([][]byte, 2+len(subs))
	for i, sub := range subs {
		if parts[2+i], e = sub.Encode(nil); e != nil {
			return nil, e
		}
	}
	return f.encodeTLVFinish(b, parts)
}

func (f Field) encodeTLVFielders(b []byte) (o []byte, e error) {
	subs := f.object.([]Fielder)
	parts := make([][]byte, 2+len(subs))
	for i, sub := range subs {
		if parts[2+i], e = sub.Field().Encode(nil); e != nil {
			return nil, e
		}
	}
	return f.encodeTLVFinish(b, parts)
}

func (f Field) encodeTLVFinish(b []byte, parts [][]byte) ([]byte, error) {
	length := 0
	for _, part := range parts {
		length += len(part)
	}

	if f.integer < math.MaxUint64 { // for EncodeValueOnly
		parts[1] = VarNum(f.integer).Encode(parts[1])
		parts[1] = VarNum(length).Encode(parts[1])
	}

	parts[0] = b
	return bytes.Join(parts, nil), nil
}

// Field implements Fielder interface.
func (f Field) Field() Field {
	return f
}

// FieldError creates a Field that generates an error.
func FieldError(e error) Field {
	if e == nil {
		e = ErrErrorField
	}
	return Field{
		typ:    fieldTypeError,
		object: e,
	}
}

// Bytes creates a Field that encodes to given bytes.
func Bytes(b []byte) Field {
	return Field{
		typ:    fieldTypeBytes,
		object: b,
	}
}

// TLV creates a Field that encodes to TLV element from TLV-TYPE and TLV-VALUE Fields.
func TLV(typ uint32, values ...Field) Field {
	if typ < minType || typ > maxType {
		return FieldError(ErrType)
	}
	return Field{
		typ:     fieldTypeTLVFields,
		integer: uint64(typ),
		object:  values,
	}
}

// TLVFrom creates a Field that encodes to TLV element from TLV-TYPE and TLV-VALUE Fielders.
func TLVFrom(typ uint32, values ...Fielder) Field {
	if typ < minType || typ > maxType {
		return FieldError(ErrType)
	}
	return Field{
		typ:     fieldTypeTLVFielders,
		integer: uint64(typ),
		object:  values,
	}
}

// TLVBytes creates a Field that encodes to TLV element from TLV-TYPE and TLV-VALUE byte slice.
func TLVBytes(typ uint32, value []byte) Field {
	return TLV(typ, Bytes(value))
}

// TLVNNI creates a Field that encodes to TLV element from TLV-TYPE and TLV-VALUE NonNegativeInteger.
func TLVNNI(typ uint32, v uint64) Field {
	return TLVFrom(typ, NNI(v))
}

// Encode encodes a sequence of Fields.
func Encode(fields ...Field) (wire []byte, e error) {
	var eb EncodingBuffer
	for _, f := range fields {
		eb.Append(f)
	}
	return eb.Output()
}

// EncodeFrom encodes a sequence of Fielders.
func EncodeFrom(fields ...Fielder) (wire []byte, e error) {
	var eb EncodingBuffer
	for _, f := range fields {
		eb.Append(f.Field())
	}
	return eb.Output()
}

// EncodeValueOnly returns TLV-VALUE of a Fielder created by TLV, TLVFrom, TLVBytes, or TLVNNI.
func EncodeValueOnly(f Fielder) ([]byte, error) {
	field := f.Field()
	switch field.typ {
	case fieldTypeError:
		return nil, field.object.(error)
	case fieldTypeTLVFields, fieldTypeTLVFielders:
		field.integer = math.MaxUint64
		return field.Encode(nil)
	default:
		return nil, ErrErrorField
	}
}
