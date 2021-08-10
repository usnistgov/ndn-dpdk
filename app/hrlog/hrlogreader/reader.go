// Package hrlogreader reads high resolution tracing logs.
package hrlogreader

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// Header constants.
const (
	Magic   = 0x35F0498A
	Version = 2
)

// Reader represents a reader for high resolution logs.
type Reader struct {
	file  *os.File
	br    *bufio.Reader
	order binary.ByteOrder

	TscHz uint64 // TSC timer frequency
}

// Read reads all entries then closes the file.
func (r *Reader) Read() <-chan uint64 {
	ch := make(chan uint64)
	go func() {
		defer func() {
			close(ch)
			r.file.Close()
		}()

		var entry [8]byte
		for {
			_, e := io.ReadFull(r.br, entry[:])
			if e != nil {
				return
			}
			ch <- r.order.Uint64(entry[:])
		}
	}()
	return ch
}

// Open opens a hrlog file.
func Open(filename string) (r *Reader, e error) {
	r = &Reader{}
	r.file, e = os.Open(filename)
	if e != nil {
		return nil, fmt.Errorf("open(%s) %w", filename, e)
	}
	defer func() {
		if e != nil {
			r.file.Close()
		}
	}()

	r.br = bufio.NewReader(r.file)

	var hdr [16]byte
	_, e = io.ReadFull(r.br, hdr[:])
	if e != nil {
		return nil, fmt.Errorf("read header %w", e)
	}

	switch {
	case binary.BigEndian.Uint32(hdr[0:4]) == Magic:
		r.order = binary.BigEndian
	case binary.LittleEndian.Uint32(hdr[0:4]) == Magic:
		r.order = binary.LittleEndian
	default:
		return nil, fmt.Errorf("bad magic %X", hdr[0:4])
	}

	if v := r.order.Uint32(hdr[4:8]); v != Version {
		return nil, fmt.Errorf("bad version %d", v)
	}

	r.TscHz = r.order.Uint64(hdr[8:16])
	return r, nil
}
