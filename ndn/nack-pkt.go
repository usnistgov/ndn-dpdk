package ndn

/*
#include "nack-pkt.h"
*/
import "C"
import (
	"fmt"
)

type NackReason uint8

const (
	NackReason_None        NackReason = 0 // packet is not a Nack
	NackReason_Congestion             = 50
	NackReason_Duplicate              = 100
	NackReason_NoRoute                = 150
	NackReason_Unspecified            = 255
)

func (nr NackReason) String() string {
	switch nr {
	case NackReason_Congestion:
		return "Congestion"
	case NackReason_Duplicate:
		return "Duplicate"
	case NackReason_NoRoute:
		return "NoRoute"
	}
	return fmt.Sprintf("%d", nr)
}
