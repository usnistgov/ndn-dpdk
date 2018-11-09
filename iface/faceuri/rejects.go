package faceuri

import (
	"fmt"
)

type rejects struct{}

var no rejects

func (rejects) user(u *FaceUri) (bool, string) {
	return u.User != nil, "user information"
}

func (rejects) password(u *FaceUri) (bool, string) {
	hasPassword := false
	if u.User != nil {
		_, hasPassword = u.User.Password()
	}
	return hasPassword, "password"
}

func (rejects) host(u *FaceUri) (bool, string) {
	return u.Host != "", "host"
}

func (rejects) port(u *FaceUri) (bool, string) {
	return u.Port() != "", "port"
}

func (rejects) path(u *FaceUri) (bool, string) {
	if u.Path == "/" {
		u.Path = ""
	}
	return u.Path != "", "path"
}

func (rejects) query(u *FaceUri) (bool, string) {
	return u.RawQuery != "", "query"
}

func (rejects) fragment(u *FaceUri) (bool, string) {
	return u.Fragment != "", "fragment"
}

// Verify that the FaceUri does not have certain fields.
func (u *FaceUri) verifyNo(rejects ...func(*FaceUri) (bool, string)) error {
	for _, reject := range rejects {
		if bad, field := reject(u); bad {
			return fmt.Errorf("should not contain %s", field)
		}
	}
	return nil
}
