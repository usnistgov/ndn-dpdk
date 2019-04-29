package iface

import (
	"encoding/json"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
)

// Identifies the endpoints of a face.
//
// Lower layer implementation must embed LocatorBase struct and provide Validate method.
// To customize serialization, implement yaml.Marshaler and yaml.Unmarshaler interfaces.
type Locator interface {
	isLocator()
	GetScheme() string

	// Check whether Locator fields are correct according to the chosen Scheme.
	Validate() error
}

// Base type to implement Locator interface.
type LocatorBase struct {
	Scheme string
}

func (LocatorBase) isLocator() {
}

func (loc LocatorBase) GetScheme() string {
	return loc.Scheme
}

// Parse Locator from YAML string.
func ParseLocator(input string) (loc Locator, e error) {
	var locw LocatorWrapper
	if e = yaml.Unmarshal([]byte(input), &locw); e != nil {
		return loc, e
	}
	loc = locw.Locator
	return loc, nil
}

func MustParseLocator(input string) (loc Locator) {
	loc, e := ParseLocator(input)
	if e != nil {
		panic(e)
	}
	return loc
}

var locatorTypes = make(map[string]reflect.Type)

// Register a Locator implementation.
func RegisterLocatorType(locator Locator, schemes ...string) {
	typ := reflect.TypeOf(locator)
	if typ.Kind() != reflect.Struct {
		panic("locator must be a struct")
	}
	for _, scheme := range schemes {
		locatorTypes[scheme] = typ
	}
}

// Wraps Locator to facilitate JSON/YAML serialization.
type LocatorWrapper struct {
	Locator
}

func (locw *LocatorWrapper) MarshalJSON() ([]byte, error) {
	obj, e := locw.MarshalYAML()
	if e != nil {
		return nil, e
	}
	return json.Marshal(obj)
}

func (locw *LocatorWrapper) UnmarshalJSON(data []byte) error {
	return locw.UnmarshalYAML(func(v interface{}) error {
		return json.Unmarshal(data, v)
	})
}

func (locw *LocatorWrapper) MarshalYAML() (interface{}, error) {
	if locw.Locator == nil {
		return nil, nil
	}

	scheme := locw.Locator.GetScheme()
	if typ, ok := locatorTypes[scheme]; !ok {
		return nil, fmt.Errorf("unknown scheme %s", scheme)
	} else if typ != reflect.TypeOf(locw.Locator) {
		return nil, fmt.Errorf("unexpected type %T", locw.Locator)
	}

	if e := locw.Locator.Validate(); e != nil {
		return nil, e
	}

	if locM, ok := locw.Locator.(yaml.Marshaler); ok {
		return locM.MarshalYAML()
	}
	return locw.Locator, nil
}

func (locw *LocatorWrapper) UnmarshalYAML(unmarshal func(interface{}) error) (e error) {
	schemeObj := struct {
		Scheme string
	}{}
	if e = unmarshal(&schemeObj); e != nil {
		return e
	}

	typ, ok := locatorTypes[schemeObj.Scheme]
	if !ok {
		return fmt.Errorf("unknown scheme %s", schemeObj.Scheme)
	}

	ptr := reflect.New(typ)
	ptrI := ptr.Interface()

	if ptrM, ok := ptrI.(yaml.Unmarshaler); ok {
		e = ptrM.UnmarshalYAML(unmarshal)
	} else {
		e = unmarshal(ptrI)
	}
	if e != nil {
		return e
	}

	loc := ptr.Elem().Interface().(Locator)
	if e = loc.Validate(); e != nil {
		return e
	}

	locw.Locator = loc
	return nil
}
