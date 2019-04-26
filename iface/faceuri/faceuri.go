package faceuri

import (
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
	return parseImpl(raw, "parse")
}

func parseImpl(raw, op string) (*FaceUri, error) {
	base, e := url.Parse(raw)
	if e != nil {
		return nil, e
	}

	if !base.IsAbs() {
		return nil, &url.Error{op, raw, errors.New("FaceUri must be absolute")}
	}

	if impl, ok := implByScheme[base.Scheme]; ok {
		u := new(FaceUri)
		u.URL = *base
		e = impl.Verify(u)
		if e != nil {
			return nil, &url.Error{op, raw, e}
		}
		return u, nil
	}

	return nil, &url.Error{op, raw, fmt.Errorf("unknown scheme %s", base.Scheme)}
}

func MustParse(raw string) *FaceUri {
	u, e := Parse(raw)
	if e != nil {
		panic(e)
	}
	return u
}

func (u *FaceUri) MarshalText() (text []byte, e error) {
	return []byte(u.String()), nil
}

func (u *FaceUri) UnmarshalText(text []byte) error {
	u2, e := Parse(string(text))
	if e == nil {
		*u = *u2
	}
	return e
}

type iImpl interface {
	// Verify a FaceUri. Update fields if necessary.
	// Return an error if FaceUri is invalid, otherwise return nil.
	Verify(u *FaceUri) error
}

var implByScheme = make(map[string]iImpl)
