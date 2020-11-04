package iface

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
)

// Locator identifies the endpoints of a face.
type Locator interface {
	// Scheme returns a string that identifies the type of this Locator.
	// Possible values must be registered through RegisterLocatorType().
	Scheme() string

	// Validate checks whether Locator fields are correct according to the chosen scheme.
	Validate() error

	// CreateFace creates a face from this Locator.
	CreateFace() (Face, error)
}

var locatorTypes = make(map[string]reflect.Type)

// RegisterLocatorType registers Locator schemes.
func RegisterLocatorType(loc Locator, schemes ...string) {
	typ := reflect.TypeOf(loc)
	if typ.Kind() != reflect.Struct {
		log.Panicf("Locator must be a struct %T", loc)
	}
	for _, scheme := range schemes {
		locatorTypes[scheme] = typ
	}
}

// LocatorWrapper wraps Locator to facilitate JSON serialization.
type LocatorWrapper struct {
	Locator
}

// MarshalJSON implements json.Marshaler.
func (locw LocatorWrapper) MarshalJSON() (data []byte, e error) {
	var m map[string]interface{}
	e = jsonhelper.Roundtrip(locw.Locator, &m)
	if e != nil {
		return nil, e
	}
	if _, ok := m["scheme"]; !ok {
		m["scheme"] = locw.Scheme()
	}
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler.
func (locw *LocatorWrapper) UnmarshalJSON(data []byte) error {
	schemeObj := struct {
		Scheme string `json:"scheme"`
	}{}
	if e := json.Unmarshal(data, &schemeObj); e != nil {
		return e
	}

	typ, ok := locatorTypes[schemeObj.Scheme]
	if !ok {
		return fmt.Errorf("unknown scheme %s", schemeObj.Scheme)
	}

	ptr := reflect.New(typ)
	if e := json.Unmarshal(data, ptr.Interface()); e != nil {
		return e
	}

	loc := ptr.Elem().Interface().(Locator)
	if e := loc.Validate(); e != nil {
		return e
	}

	locw.Locator = loc
	return nil
}

// LocatorString converts a locator to JSON string
func LocatorString(loc Locator) string {
	locw := LocatorWrapper{Locator: loc}
	j, _ := json.Marshal(locw)
	return string(j)
}
