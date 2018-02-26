package ndn

/*
#include "data.h"
*/
import "C"
import "time"

// Data packet.
type Data struct {
	m Packet
	p *C.PData
}

func (data *Data) GetPacket() Packet {
	return data.m
}

func (data *Data) GetName() (n *Name) {
	n = new(Name)
	n.copyFromC(&data.p.name)
	return n
}

func (data *Data) GetFreshnessPeriod() time.Duration {
	return time.Duration(data.p.freshnessPeriod) * time.Millisecond
}
