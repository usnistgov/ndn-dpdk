package ndn6file

import (
	"bytes"
	"encoding"
	"fmt"
	"io/fs"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

// KeywordLs is the 32=ls component.
var KeywordLs = ndn.MakeNameComponent(an.TtKeywordNameComponent, []byte("ls"))

// DirectoryListing contains a list of files and directories in a directory.
type DirectoryListing []fs.DirEntry

var (
	_ encoding.BinaryMarshaler   = DirectoryListing{}
	_ encoding.BinaryUnmarshaler = (*DirectoryListing)(nil)
)

// MarshalBinary encodes to segmented object payload.
func (ls DirectoryListing) MarshalBinary() (value []byte, e error) {
	var b bytes.Buffer
	for _, f := range ls {
		b.WriteString(f.Name())
		if f.IsDir() {
			b.WriteByte('/')
		}
		b.WriteByte(0)
	}
	return b.Bytes(), nil
}

// UnmarshalBinary decodes from segmented object payload.
func (ls *DirectoryListing) UnmarshalBinary(value []byte) (e error) {
	totalLen := len(value)
	*ls = DirectoryListing{}
	for len(value) > 0 {
		pos := bytes.IndexByte(value, 0)
		if pos <= 0 {
			return fmt.Errorf("unterminated or blank line near offset %d", totalLen-len(value))
		}
		if value[pos-1] == '/' {
			*ls = append(*ls, directoryEntry{
				name:  string(value[:pos-1]),
				isDir: true,
			})
		} else {
			*ls = append(*ls, directoryEntry{
				name:  string(value[:pos]),
				isDir: false,
			})
		}
		value = value[pos+1:]
	}
	return nil
}

type directoryEntry struct {
	name  string
	isDir bool
}

func (f directoryEntry) Name() string {
	return f.name
}

func (f directoryEntry) IsDir() bool {
	return f.isDir
}

func (f directoryEntry) Type() fs.FileMode {
	if f.isDir {
		return fs.ModeDir
	}
	return 0
}

func (f directoryEntry) Info() (fs.FileInfo, error) {
	return f, nil
}

func (directoryEntry) Size() int64 {
	return -1
}

func (f directoryEntry) Mode() fs.FileMode {
	return f.Type() | fs.ModePerm
}

func (directoryEntry) ModTime() time.Time {
	return time.Time{}
}

func (directoryEntry) Sys() any {
	return nil
}
