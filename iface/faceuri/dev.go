package faceuri

import "errors"

type devImpl struct{}

func (devImpl) Verify(u *FaceUri) error {
	e := rejectUPQF(u)
	if e != nil {
		return e
	}

	if u.Port() != "" {
		return errors.New("dev URI cannot have port number")
	}

	return nil
}

func init() {
	implByScheme["dev"] = devImpl{}
}
