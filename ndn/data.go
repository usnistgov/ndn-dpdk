package ndn

/*
#include "data.h"
*/
import "C"
import (
	"time"
	"unsafe"
)

// Data packet.
type Data struct {
	m Packet
	p *C.PData
}

func (data *Data) GetPacket() Packet {
	return data.m
}

func (data *Data) String() string {
	return data.GetName().String()
}

// Get *C.PData pointer.
func (data *Data) GetPDataPtr() unsafe.Pointer {
	return unsafe.Pointer(data.p)
}

func (data *Data) GetName() (n *Name) {
	n = new(Name)
	n.copyFromC(&data.p.name)
	return n
}

func (data *Data) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.p.freshnessPeriod) * time.Millisecond
}
