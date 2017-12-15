package ndn

/*
#include "data-pkt.h"
*/
import "C"
import (
	"time"
	"unsafe"
)

type DataPkt struct {
	c C.DataPkt
}

// Decode a Data.
func (d *TlvDecoder) ReadData() (data DataPkt, e error) {
	res := C.DecodeData(d.getPtr(), &data.c)
	if res != C.NdnError_OK {
		return DataPkt{}, NdnError(res)
	}
	return data, nil
}

func (data *DataPkt) GetName() *Name {
	return (*Name)(unsafe.Pointer(&data.c.name))
}

func (data *DataPkt) ComputeDigest() []byte {
	panic("not implemented")
}

func (data *DataPkt) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.c.freshnessPeriod) * time.Millisecond
}
