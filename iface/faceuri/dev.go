package faceuri

type devImpl struct{}

func (devImpl) Verify(u *FaceUri) error {
	if e := u.verifyNo(no.user, no.port, no.path, no.query, no.fragment); e != nil {
		return e
	}

	return nil
}

func init() {
	implByScheme["dev"] = devImpl{}
}
