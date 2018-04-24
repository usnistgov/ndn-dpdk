package faceuri

import (
	"errors"
	"path/filepath"
)

type unixImpl struct{}

func (unixImpl) Verify(u *FaceUri) error {
	if e := u.verifyNo(no.user, no.host, no.query, no.fragment); e != nil {
		return e
	}

	u.Path = filepath.Clean(u.Path)

	if u.Path[0] != '/' {
		return errors.New("unix FaceUri must have absolute filesystem path")
	}

	return nil
}

func init() {
	implByScheme["unix"] = unixImpl{}
}
