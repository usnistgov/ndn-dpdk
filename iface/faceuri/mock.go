package faceuri

import "errors"

type mockImpl struct{}

func (mockImpl) Verify(u *FaceUri) error {
	if u.String() != "mock://" {
		return errors.New("mock URI must be 'mock://'")
	}
	return nil
}

func init() {
	implByScheme["mock"] = mockImpl{}
}
