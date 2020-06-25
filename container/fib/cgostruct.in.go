// +build ignore

package fib

/*
#include "../../csrc/fib/entry-struct.h"
*/
import "C"

// CEntry is the FIB entry representation in C.
type CEntry C.FibEntry
