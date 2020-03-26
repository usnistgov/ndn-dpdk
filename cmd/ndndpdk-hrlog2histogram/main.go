package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"io"
	"os"
	"time"
)

var (
	r      *os.File
	order  binary.ByteOrder
	tschz  uint64
	tscMul float64 // TSC*tscMul = microseconds
)

func main() {
	var filename string
	flag.StringVar(&filename, "f", "", "input .hrlog filename")
	flag.Parse()

	var e error
	if r, e = os.Open(filename); e != nil {
		panic(e)
	}
	defer r.Close()
	if e = readHeader(); e != nil {
		panic(e)
	}
	tscMul = float64(time.Second) / float64(time.Microsecond) / float64(tschz)

	hists := readEntries()
	var histArray []*histogram
	for _, hist := range hists {
		hist.Trim()
		histArray = append(histArray, hist)
	}
	json.NewEncoder(os.Stdout).Encode(histArray)
}

func readHeader() (e error) {
	var hdr [16]byte
	if _, e = r.ReadAt(hdr[:], 0); e != nil {
		return
	}

	const magic = 0x35f0498a
	if binary.LittleEndian.Uint32(hdr[0:]) == magic {
		order = binary.LittleEndian
	} else if binary.BigEndian.Uint32(hdr[0:]) == magic {
		order = binary.BigEndian
	} else {
		panic("invalid magic number")
	}

	version := order.Uint32(hdr[4:])
	if version != 2 {
		panic("invalid file version")
	}

	tschz = order.Uint64(hdr[8:])
	return
}

func readEntries() (hists map[uint16]*histogram) {
	hists = make(map[uint16]*histogram)
	var buf [4096]byte
	for off := int64(16); ; off += int64(len(buf)) {
		n, e := r.ReadAt(buf[:], off)
		if e != nil && e != io.EOF {
			panic(e)
		}

		for pos := 0; pos < n; pos += 8 {
			entry := order.Uint64(buf[pos:])
			lcoreAct := uint16(entry)
			value := entry >> 16
			microseconds := float64(value) * tscMul

			hist := hists[lcoreAct]
			if hist == nil {
				hist = newHistogram(uint8(lcoreAct), uint8(lcoreAct>>8))
				hists[lcoreAct] = hist
			}
			hist.Add(int(microseconds))
		}

		if e != nil {
			break
		}
	}
	return hists
}
