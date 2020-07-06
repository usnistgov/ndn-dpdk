package pingmgmt

import (
	"errors"
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/ping"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type FetchMgmt struct {
	App *ping.App
}

func (mg FetchMgmt) getFetcher(index int) (fetcher *fetch.Fetcher, e error) {
	if index >= len(mg.App.Tasks) {
		return nil, errors.New("Index out of range")
	}
	fetcher = mg.App.Tasks[index].Fetch
	if fetcher == nil {
		return nil, errors.New("Task has no Fetcher")
	}
	return fetcher, nil
}

func (mg FetchMgmt) List(args struct{}, reply *[]int) error {
	var list []int
	for index, task := range mg.App.Tasks {
		if task.Fetch != nil {
			list = append(list, index)
		}
	}
	*reply = list
	return nil
}

func (mg FetchMgmt) Benchmark(args FetchBenchmarkArgs, reply *[]FetchBenchmarkReply) error {
	fetcher, e := mg.getFetcher(args.Index)
	if e != nil {
		return e
	}
	if fetcher.Thread(0).IsRunning() {
		return errors.New("Fetcher is running")
	}

	var logics []*fetch.Logic
	list := make([]FetchBenchmarkReply, len(args.Templates))
	fetcher.Reset()
	for i, tpl := range args.Templates {
		tplArgs := []interface{}{tpl.Prefix}
		if tpl.CanBePrefix {
			tplArgs = append(tplArgs, ndn.CanBePrefixFlag)
		}
		if d := tpl.InterestLifetime.Duration(); d > 0 {
			tplArgs = append(tplArgs, d)
		}
		if j, e := fetcher.AddTemplate(tplArgs...); e != nil {
			return fmt.Errorf("AddTemplate[%d]: %s", i, e)
		} else {
			logics = append(logics, fetcher.Logic(j))
		}
	}

	fetcher.Launch()
	ticker := time.NewTicker(args.Interval.Duration())
	defer ticker.Stop()
	for c := 0; c < args.Count; c++ {
		<-ticker.C
		for i, logic := range logics {
			list[i].Counters = append(list[i].Counters, logic.ReadCounters())
		}
	}

	fetcher.Stop()
	for i, rp := range list {
		list[i].Goodput = rp.Counters[len(rp.Counters)-1].ComputeGoodput(rp.Counters[0])
	}
	*reply = list
	return nil
}

type FetchTemplate struct {
	Prefix           ndn.Name
	InterestLifetime nnduration.Milliseconds
	CanBePrefix      bool
}

type FetchBenchmarkArgs struct {
	IndexArg
	Templates []FetchTemplate
	Interval  nnduration.Milliseconds
	Count     int
}

type FetchBenchmarkReply struct {
	Counters []fetch.Counters
	Goodput  float64
}
