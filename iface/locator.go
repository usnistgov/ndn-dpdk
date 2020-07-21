package iface

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Locator identifies the endpoints of a face.
type Locator interface {
	// Scheme returns a string that identifies the type of this Locator.
	// Possible values must be registered through RegisterLocatorType().
	Scheme() string

	// Validate checks whether Locator fields are correct according to the chosen scheme.
	Validate() error
}

// ParseLocator parses Locator from JSON string.
func ParseLocator(input string) (loc Locator, e error) {
	var locw LocatorWrapper
	if e = json.Unmarshal([]byte(input), &locw); e != nil {
		return loc, e
	}
	loc = locw.Locator
	return loc, nil
}

// MustParseLocator parses Locator from JSON string, and panics on error.
func MustParseLocator(input string) (loc Locator) {
	loc, e := ParseLocator(input)
	if e != nil {
		log.Panic("bad Locator", input, e)
	}
	return loc
}

var locatorTypes = make(map[string]reflect.Type)

// RegisterLocatorType registers Locator schemes.
func RegisterLocatorType(locator Locator, schemes ...string) {
	typ := reflect.TypeOf(locator)
	if typ.Kind() != reflect.Struct {
		log.Panicf("locator must be a struct %T", locator)
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
	return json.Marshal(locw.Locator)
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
