package ndn

/*
#include "nack-pkt.h"
*/
import "C"

type NackReason uint8

const (
	NackReason_None        NackReason = 0 // packet is not a Nack
	NackReason_Congestion             = 50
	NackReason_Duplicate              = 100
	NackReason_NoRoute                = 150
	NackReason_Unspecified            = 255
)
