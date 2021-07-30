// Command ndndpdk-hrlog2histogram extracts latency histograms from high resolution per-packet logs.
package main

import (
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/hrlog/hrlogreader"
)

func main() {
	var filename string
	flag.StringVar(&filename, "f", "", "input .hrlog filename")
	flag.Parse()

	r, e := hrlogreader.Open(filename)
	if e != nil {
		panic(e)
	}
	tscMul := float64(time.Second) / float64(time.Microsecond) / float64(r.TscHz)

	hists := map[uint16]*histogram{}
	for entry := range r.Read() {
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

	var histArray []*histogram
	for _, hist := range hists {
		hist.Trim()
		histArray = append(histArray, hist)
	}
	json.NewEncoder(os.Stdout).Encode(histArray)
}
