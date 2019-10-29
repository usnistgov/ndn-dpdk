package fetch

/*
#include "rttest.h"
*/
import "C"
import (
	"ndn-dpdk/dpdk"
)

type RttEst struct {
	c *C.RttEst
}

func NewRttEst() (rtte *RttEst) {
	rtte = new(RttEst)
	rtte.c = new(C.RttEst)
	C.RttEst_Init(rtte.c)
	return rtte
}

func (rtte *RttEst) GetRtt() int64 {
	return int64(rtte.c.rtt)
}

func (rtte *RttEst) GetRto() int64 {
	return int64(rtte.c.rto)
}

func (rtte *RttEst) Push(since, now dpdk.TscTime) {
	C.RttEst_Push(rtte.c, C.TscTime(since), C.TscTime(now))
}

func (rtte *RttEst) Backoff() {
	C.RttEst_Backoff(rtte.c)
}
