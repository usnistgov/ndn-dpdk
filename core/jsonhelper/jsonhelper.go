// Package jsonhelper provides JSON-related helper functions.
package jsonhelper

import (
	"bytes"
	"encoding/json"
)

// Option sets an option on json.Decoder.
type Option func(*json.Decoder)

// DisallowUnknownFields causes json.Decoder to reject unknown struct fields.
var DisallowUnknownFields Option = func(d *json.Decoder) { d.DisallowUnknownFields() }

// Roundtrip marshals the input to JSON then unmarshals it into ptr.
// This is useful for converting between structures.
func Roundtrip(input, ptr any, options ...Option) error {
	j, e := json.Marshal(input)
	if e != nil {
		return e
	}

	decoder := json.NewDecoder(bytes.NewReader(j))
	for _, option := range options {
		option(decoder)
	}
	e = decoder.Decode(ptr)
	if e != nil {
		return e
	}
	return nil
}
