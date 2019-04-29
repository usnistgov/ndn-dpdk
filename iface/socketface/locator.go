package socketface

import (
	"fmt"

	"ndn-dpdk/iface"
)

type Locator struct {
	iface.LocatorBase `yaml:",inline"`
	Local             string
	Remote            string
}

func (loc Locator) Validate() error {
	impl, ok := implByNetwork[loc.Scheme]
	if !ok {
		return fmt.Errorf("unknown scheme %s", loc.Scheme)
	}

	if loc.Local != "" {
		if e := impl.ValidateAddr(loc.Scheme, loc.Local, true); e != nil {
			return fmt.Errorf("Local: %v", e)
		}
	}
	if e := impl.ValidateAddr(loc.Scheme, loc.Remote, false); e != nil {
		return fmt.Errorf("Remote: %v", e)
	}
	return nil
}

func Create(loc Locator, cfg Config) (face *SocketFace, e error) {
	if e = loc.Validate(); e != nil {
		return nil, e
	}

	impl := implByNetwork[loc.Scheme]
	conn, e := impl.Dial(loc.Scheme, loc.Local, loc.Remote)
	if e != nil {
		return nil, e
	}
	return New(conn, cfg)
}
