package testenv

/*
#include <stdlib.h>
*/
import "C"

func Exit(code int) {
	C.exit(C.int(code))
}
