package faceuri

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
)

type FaceUri struct {
	url.URL
}

func (u *FaceUri) String() string {
	return u.URL.String()
}

func Parse(raw string) (*FaceUri, error) {
	base, e := url.Parse(raw)
	if e != nil {
		return nil, e
	}

	if !base.IsAbs() {
		return nil, errors.New("FaceUri must be absolute")
	}

	if impl, ok := implByScheme[base.Scheme]; ok {
		u := new(FaceUri)
		u.URL = *base
		e = impl.Verify(u)
		if e != nil {
			return nil, e
		}
		return u, nil
	}

	return nil, fmt.Errorf("unknown scheme %s", base.Scheme)
}

func MustParse(raw string) *FaceUri {
	u, e := Parse(raw)
	if e != nil {
		panic(e)
	}
	return u
}

func (u *FaceUri) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *FaceUri) UnmarshalJSON(data []byte) error {
	return u.UnmarshalYAML(func(v interface{}) error {
		return json.Unmarshal(data, v)
	})
}

func (u *FaceUri) MarshalYAML() (interface{}, error) {
	return u.String(), nil
}

func (u *FaceUri) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	if e := unmarshal(&raw); e != nil {
		return e
	}
	if u2, e := Parse(raw); e != nil {
		return e
	} else {
		*u = *u2
	}
	return nil
}

type iImpl interface {
	// Verify a FaceUri. Update fields if necessary.
	// Return an error if FaceUri is invalid, otherwise return nil.
	Verify(u *FaceUri) error
}

var implByScheme = make(map[string]iImpl)
