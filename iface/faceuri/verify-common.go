package faceuri

import (
	"fmt"
)

// Error if FaceUri contains User, Path, Query, or Fragment.
func rejectUPQF(u *FaceUri) error {
	if u.User != nil {
		return fmt.Errorf("%s URI cannot have user information", u.Scheme)
	}
	if u.Path != "" {
		if u.Path == "/" {
			u.Path = ""
		} else {
			return fmt.Errorf("%s URI cannot have path", u.Scheme)
		}
	}
	if u.RawQuery != "" {
		return fmt.Errorf("%s URI cannot have query", u.Scheme)
	}
	if u.Fragment != "" {
		return fmt.Errorf("%s URI cannot have fragment", u.Scheme)
	}
	return nil
}
