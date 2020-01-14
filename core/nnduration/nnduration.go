package nnduration

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

func parse(input string, unit time.Duration) (value uint64, e error) {
	if d, e := time.ParseDuration(input); e == nil {
		return uint64(d / unit), nil
	}
	return strconv.ParseUint(input, 10, 64)
}

func parseJson(ptr interface{}, p []byte, unit time.Duration) error {
	value, e := parse(strings.Trim(string(p), `"`), unit)
	reflect.ValueOf(ptr).Elem().SetUint(value)
	return e
}

type Milliseconds uint64

func (d *Milliseconds) UnmarshalJSON(p []byte) (e error) {
	return parseJson(d, p, time.Millisecond)
}

func (d Milliseconds) Duration() time.Duration {
	return time.Duration(d) * time.Millisecond
}

type Nanoseconds uint64

func (d *Nanoseconds) UnmarshalJSON(p []byte) (e error) {
	return parseJson(d, p, time.Nanosecond)
}

func (d Nanoseconds) Duration() time.Duration {
	return time.Duration(d) * time.Nanosecond
}
