package cptr

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"errors"
	"io"
	"os"
	"unsafe"

	"go.uber.org/multierr"
	"golang.org/x/sys/unix"
)

// FilePipeConfig configures FilePipe*.
type FilePipeConfig struct {
	// NonBlock sets O_NONBLOCK on the writer file descriptor.
	NonBlock bool
}

// FilePipeCGo is a pipe from *C.FILE writer to *os.File reader.
type FilePipeCGo struct {
	Reader *os.File
	Writer unsafe.Pointer
}

// ReadAll reads from the pipe until EOF.
func (p *FilePipeCGo) ReadAll() (data []byte, e error) {
	if p.Reader == nil {
		return nil, io.ErrClosedPipe
	}
	return io.ReadAll(p.Reader)
}

// CloseReader closes the reader.
func (p *FilePipeCGo) CloseReader() (e error) {
	if p.Reader != nil {
		e = p.Reader.Close()
		p.Reader = nil
	}
	return e
}

// CloseWriter closes the writer.
func (p *FilePipeCGo) CloseWriter() error {
	if p.Writer != nil {
		C.fclose((*C.FILE)(p.Writer))
		p.Writer = nil
	}
	return nil
}

// Close closes both reader and writer.
func (p *FilePipeCGo) Close() error {
	return multierr.Append(
		p.CloseWriter(),
		p.CloseReader(),
	)
}

// NewFilePipeCGo creates a FilePipeCGo.
func NewFilePipeCGo(cfg FilePipeConfig) (p *FilePipeCGo, e error) {
	pipefd := make([]int, 2)
	if e := unix.Pipe(pipefd); e != nil {
		return nil, e
	}
	defer func() {
		for _, fd := range pipefd {
			unix.Close(fd)
		}
	}()

	if cfg.NonBlock {
		if e = unix.SetNonblock(pipefd[1], true); e != nil {
			return nil, errors.New("unix.SetNonblock error")
		}
	}

	p = &FilePipeCGo{}

	wMode := []C.char{'w', 0}
	p.Writer = unsafe.Pointer(C.fdopen(C.int(pipefd[1]), &wMode[0]))
	if p.Writer == nil {
		return nil, errors.New("fdopen error")
	}

	p.Reader = os.NewFile(uintptr(pipefd[0]), "")
	if p.Reader == nil {
		return nil, errors.New("os.NewFile error")
	}

	pipefd = nil
	return p, nil
}

// CaptureFileDump invokes a function that writes to *C.FILE, and returns what's been written.
func CaptureFileDump(f func(fp unsafe.Pointer)) (data []byte, e error) {
	p, e := NewFilePipeCGo(FilePipeConfig{})
	if e != nil {
		return nil, e
	}
	defer p.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		data, e = p.ReadAll()
	}()

	f(p.Writer)
	p.CloseWriter()
	<-done
	return
}
