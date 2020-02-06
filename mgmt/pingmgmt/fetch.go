package pingmgmt

import (
	"errors"
	"time"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/ping"
	"ndn-dpdk/core/nnduration"
	"ndn-dpdk/ndn"
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

func (mg FetchMgmt) Benchmark(args FetchBenchmarkArgs, reply *FetchBenchmarkReply) error {
	fetcher, e := mg.getFetcher(args.Index)
	if e != nil {
		return e
	}
	if fetcher.IsRunning() {
		return errors.New("Fetcher is running")
	}

	fetcher.SetName(args.Name)
	fetcher.Launch()
	time.Sleep(args.Warmup.Duration())
	firstCnt := fetcher.Logic.ReadCounters()
	cnt := firstCnt

	ticker := time.NewTicker(args.Interval.Duration())
	defer ticker.Stop()
	for len(reply.Counters) < args.Count {
		<-ticker.C
		if !fetcher.IsRunning() {
			return errors.New("Fetcher stopped prematurely")
		}
		cnt = fetcher.Logic.ReadCounters()
		reply.Counters = append(reply.Counters, cnt)
	}
	reply.Goodput = cnt.ComputeGoodput(firstCnt)
	fetcher.Stop()
	return nil
}

func (mg FetchMgmt) ReadCounters(args IndexArg, reply *fetch.Counters) error {
	fetcher, e := mg.getFetcher(args.Index)
	if e != nil {
		return e
	}

	*reply = fetcher.Logic.ReadCounters()
	return nil
}

type FetchBenchmarkArgs struct {
	Index    int // Task index
	Name     *ndn.Name
	Warmup   nnduration.Milliseconds
	Interval nnduration.Milliseconds
	Count    int
}

type FetchBenchmarkReply struct {
	Counters []fetch.Counters
	Goodput  float64
}
