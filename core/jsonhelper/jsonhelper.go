// Package jsonhelper provides JSON-related helper functions.
package jsonhelper

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/pascaldekloe/name"
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

// CleanCgoStruct cleans roundtripped object from a cgo-generated structure.
// It deletes padding and reserved fields, then converts object keys to lowerCamalCase.
func CleanCgoStruct(input any) any {
	switch input := input.(type) {
	case map[string]any:
		m := map[string]any{}
		for k, v := range input {
			if strings.HasPrefix(k, "Pad_cgo_") || strings.HasPrefix(k, "Reserved_") {
				continue
			}
			m[name.CamelCase(k, false)] = CleanCgoStruct(v)
		}
		return m
	case []any:
		a := make([]any, len(input))
		for i, v := range input {
			a[i] = CleanCgoStruct(v)
		}
		return a
	default:
		return input
	}
}
