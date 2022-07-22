package tlv

import (
	"math"

	"golang.org/x/exp/constraints"
)

// EncodingBuffer is an encoding buffer.
// Zero value is an empty buffer.
type EncodingBuffer struct {
	b   []byte
	err error
}

// Append appends a field.
// If there's an error, it is accumulated in the EncodingBuffer.
func (eb *EncodingBuffer) Append(f Field) {
	if eb.err != nil {
		return
	}
	eb.b, eb.err = f.Encode(eb.b)
}

// Output returns encoding output and accumulated error.
func (eb EncodingBuffer) Output() ([]byte, error) {
	return eb.b, eb.err
}

type fieldType uint8

const (
	fieldTypeEmpty fieldType = iota
	fieldTypeError
	fieldTypeFunc
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
	object  any
}

// Encode appends to the byte slice.
// Returns modified slice and error.
func (f Field) Encode(b []byte) ([]byte, error) {
	switch f.typ {
	case fieldTypeEmpty:
		return b, nil
	case fieldTypeError:
		return nil, f.object.(error)
	case fieldTypeFunc:
		return f.object.(func([]byte) ([]byte, error))(b)
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
	parts := make([][]byte, len(subs))
	for i, sub := range subs {
		if parts[i], e = sub.Encode(nil); e != nil {
			return nil, e
		}
	}
	return f.encodeTLVFinish(b, parts)
}

func (f Field) encodeTLVFielders(b []byte) (o []byte, e error) {
	subs := f.object.([]Fielder)
	parts := make([][]byte, len(subs))
	for i, sub := range subs {
		if parts[i], e = sub.Field().Encode(nil); e != nil {
			return nil, e
		}
	}
	return f.encodeTLVFinish(b, parts)
}

func (f Field) encodeTLVFinish(b []byte, value [][]byte) ([]byte, error) {
	length := 0
	for _, part := range value {
		length += len(part)
	}

	if f.integer < math.MaxUint64 { // not EncodeValueOnly
		b = VarNum(f.integer).Encode(b)
		b = VarNum(length).Encode(b)
	}

	for _, part := range value {
		b = append(b, part...)
	}
	return b, nil
}

// Field implements Fielder interface.
func (f Field) Field() Field {
	return f
}

// Fielder is the interface implemented by an object that can encode itself to a Field.
type Fielder interface {
	Field() Field
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

// FieldFunc creates a Field that calls a function to append to a slice.
func FieldFunc(f func([]byte) ([]byte, error)) Field {
	return Field{
		typ:    fieldTypeFunc,
		object: f,
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
func TLVNNI[V constraints.Integer](typ uint32, v V) Field {
	if v < 0 {
		return FieldError(ErrRange)
	}
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
