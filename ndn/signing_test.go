package ndn_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestDigestSigning(t *testing.T) {
	var c ndntestenv.SignVerifyTester
	c.PvtA, c.PvtB, c.PubA, c.PubB = ndn.DigestSigning, ndn.DigestSigning, ndn.DigestSigning, ndn.DigestSigning
	c.SameAB = true
	c.CheckInterest(t)
	c.CheckInterestParameterized(t)
	c.CheckData(t)
}

func TestNullSigner(t *testing.T) {
	var c ndntestenv.SignVerifyTester
	c.PvtA, c.PvtB, c.PubA, c.PubB = ndn.NullSigner, ndn.NullSigner, ndn.NopVerifier, ndn.NopVerifier
	c.SameAB = true
	c.CheckInterest(t)
	c.CheckInterestParameterized(t)
	c.CheckData(t)
}
