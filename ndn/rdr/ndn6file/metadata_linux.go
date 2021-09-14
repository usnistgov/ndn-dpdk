package ndn6file

import "golang.org/x/sys/unix"

func _() {
	var x [1]byte
	_ = x[unix.S_IFREG-sIFREG]
	_ = x[unix.S_IFDIR-sIFDIR]
}
