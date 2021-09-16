//go:build ignore

package fileserver

/*
#include "../../csrc/fileserver/server.h"
*/
import "C"

type countersC C.FileServerCounters
