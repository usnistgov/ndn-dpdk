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

func (mg FetchMgmt) getFetcher(index int, fetchId int) (fetcher *fetch.Fetcher, e error) {
	if index >= len(mg.App.Tasks) {
		return nil, errors.New("Index out of range")
	}
	fetchers := mg.App.Tasks[index].Fetch
	if fetchId >= len(fetchers) {
		return nil, errors.New("FetchId out of range")
	}
	return fetchers[fetchId], nil
}

func (mg FetchMgmt) List(args struct{}, reply *[]FetchIndexArg) error {
	var list []FetchIndexArg
	for index, task := range mg.App.Tasks {
		for fetchId := range task.Fetch {
			var item FetchIndexArg
			item.Index = index
			item.FetchId = fetchId
			list = append(list, item)
		}
	}
	*reply = list
	return nil
}

func (mg FetchMgmt) Benchmark(args FetchBenchmarkArgs, reply *FetchBenchmarkReply) error {
	fetcher, e := mg.getFetcher(args.Index, args.FetchId)
	if e != nil {
		return e
	}
	if fetcher.IsRunning() {
		return errors.New("Fetcher is running")
	}

	fetcher.Logic.Reset()
	fetcher.SetNames(args.Names)
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

func (mg FetchMgmt) ReadCounters(args FetchIndexArg, reply *fetch.Counters) error {
	fetcher, e := mg.getFetcher(args.Index, args.FetchId)
	if e != nil {
		return e
	}

	*reply = fetcher.Logic.ReadCounters()
	return nil
}

type FetchIndexArg struct {
	IndexArg
	FetchId int
}

type FetchBenchmarkArgs struct {
	FetchIndexArg
	Names    []*ndn.Name
	Warmup   nnduration.Milliseconds
	Interval nnduration.Milliseconds
	Count    int
}

type FetchBenchmarkReply struct {
	Counters []fetch.Counters
	Goodput  float64
}
