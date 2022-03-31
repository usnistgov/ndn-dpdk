// Package nnduration provides JSON-friendly non-negative duration types.
package nnduration

import (
	"bytes"
	"reflect"
	"strconv"
	"time"
)

func parse(input string, unit time.Duration) (value uint64, e error) {
	if d, e := time.ParseDuration(input); e == nil {
		return uint64(d / unit), nil
	}
	return strconv.ParseUint(input, 10, 64)
}

func parseJSON(ptr any, p []byte, unit time.Duration) error {
	value, e := parse(string(bytes.Trim(p, `"`)), unit)
	reflect.ValueOf(ptr).Elem().SetUint(value)
	return e
}

// Milliseconds is a duration in milliseconds unit.
type Milliseconds uint64

// UnmarshalJSON implements json.Unmarshaler interface.
// It accepts either an integer as milliseconds, or a duration string recognized by time.ParseDuration.
func (d *Milliseconds) UnmarshalJSON(p []byte) (e error) {
	return parseJSON(d, p, time.Millisecond)
}

// Duration converts this to time.Duration.
func (d Milliseconds) Duration() time.Duration {
	return time.Duration(d) * time.Millisecond
}

// DurationOr converts this to time.Duration, but returns dflt if this is zero.
func (d Milliseconds) DurationOr(dflt Milliseconds) time.Duration {
	if d == 0 {
		return dflt.Duration()
	}
	return d.Duration()
}

// Nanoseconds is a duration in nanoseconds unit.
type Nanoseconds uint64

// UnmarshalJSON implements json.Unmarshaler interface.
// It accepts either an integer as nanoseconds, or a duration string recognized by time.ParseDuration.
func (d *Nanoseconds) UnmarshalJSON(p []byte) (e error) {
	return parseJSON(d, p, time.Nanosecond)
}

// Duration converts this to time.Duration.
func (d Nanoseconds) Duration() time.Duration {
	return time.Duration(d) * time.Nanosecond
}

// DurationOr converts this to time.Duration, but returns dflt if this is zero.
func (d Nanoseconds) DurationOr(dflt Nanoseconds) time.Duration {
	if d == 0 {
		return dflt.Duration()
	}
	return d.Duration()
}
