package ndnitest

/*
#include "../../csrc/ndni/name.h"

typedef struct PNameUnpacked
{
	int16_t firstNonGeneric;
	bool hasDigestComp;
} PNameUnpacked;

static void
c_PName_Unpack(const PName* p, PNameUnpacked* u)
{
	u->firstNonGeneric = p->firstNonGeneric;
	u->hasDigestComp = p->hasDigestComp;
}
*/
import "C"
import (
	"crypto/sha256"
	"math"
	"strings"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func checkPName(t *testing.T, name string, f func(p *C.PName, u *C.PNameUnpacked)) {
	assert, _ := makeAR(t)

	n := ndn.ParseName(name)
	pn := ndni.NewPName(n)
	defer pn.Free()

	p := (*C.PName)(pn.Ptr())
	var u C.PNameUnpacked
	C.c_PName_Unpack(p, &u)

	assert.EqualValues(n.Length(), p.length)
	assert.EqualValues(len(n), p.nComps)
	f(p, &u)
}

func fromLName(l C.LName) (n ndn.Name) {
	wire := C.GoBytes(unsafe.Pointer(l.value), C.int(l.length))
	if e := n.UnmarshalBinary(wire); e != nil {
		return ndn.Name{}
	}
	return n
}

func ctestPName(t *testing.T) {
	assert, _ := makeAR(t)

	checkPName(t, "/", func(p *C.PName, u *C.PNameUnpacked) {
		assert.EqualValues(-1, u.firstNonGeneric)
		assert.EqualValues(false, u.hasDigestComp)
	})

	checkPName(t, "/A/B", func(p *C.PName, u *C.PNameUnpacked) {
		assert.EqualValues(-1, u.firstNonGeneric)
		assert.EqualValues(false, u.hasDigestComp)

		nameEqual(assert, "/", fromLName(C.PName_GetPrefix(p, -3)))
		nameEqual(assert, "/", fromLName(C.PName_GetPrefix(p, -2)))
		nameEqual(assert, "/A", fromLName(C.PName_GetPrefix(p, 1)))
		nameEqual(assert, "/", fromLName(C.PName_GetPrefix(p, 0)))
		nameEqual(assert, "/A", fromLName(C.PName_GetPrefix(p, 1)))
		nameEqual(assert, "/A/B", fromLName(C.PName_GetPrefix(p, 2)))
		nameEqual(assert, "/A/B", fromLName(C.PName_GetPrefix(p, 3)))
	})

	checkPName(t, "/A/1="+strings.Repeat("%00", sha256.Size), func(p *C.PName, u *C.PNameUnpacked) {
		assert.EqualValues(1, u.firstNonGeneric)
		assert.EqualValues(true, u.hasDigestComp)
	})

	checkPName(t, "/A/B/1=C/253=D/256=E/65535=F/G/32=metadata/35=%01/33=%02", func(p *C.PName, u *C.PNameUnpacked) {
		assert.EqualValues(2, u.firstNonGeneric)
		assert.EqualValues(false, u.hasDigestComp)

		nameEqual(assert, "/1=C", fromLName(C.PName_Slice(p, 2, 3)))
		nameEqual(assert, "/253=D", fromLName(C.PName_Slice(p, 3, 4)))
		nameEqual(assert, "/256=E", fromLName(C.PName_Slice(p, 4, 5)))
		nameEqual(assert, "/65535=F", fromLName(C.PName_Slice(p, 5, 6)))
	})

	checkPName(t, "/A/1="+strings.Repeat("%00", sha256.Size-1), func(p *C.PName, u *C.PNameUnpacked) {
		assert.EqualValues(1, u.firstNonGeneric)
		assert.EqualValues(false, u.hasDigestComp) // wrong TLV-LENGTH
	})

	nameAZ := "/A/B/C/D/E/F/G/H/I/J/K/L/M/N/O/P/Q/R/S/T/U/V/W/X/Y/Z"
	checkPName(t, nameAZ, func(p *C.PName, u *C.PNameUnpacked) {
		assert.EqualValues(-1, u.firstNonGeneric)
		assert.EqualValues(false, u.hasDigestComp)

		for i := 0; i <= 26; i++ {
			nameEqual(assert, nameAZ[:i*2], fromLName(C.PName_GetPrefix(p, C.int16_t(i))))
			nameEqual(assert, nameAZ[i*2:], fromLName(C.PName_Slice(p, C.int16_t(i), math.MaxInt16)))
		}
		nameEqual(assert, nameAZ[4:48], fromLName(C.PName_Slice(p, 2, -2)))
	})
}
