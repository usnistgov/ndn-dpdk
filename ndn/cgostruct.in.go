// +build ignore

package ndn

/*
#include "interest.h"
*/
import "C"

// Template to encode an Interest.
type InterestTemplate C.InterestTemplate
