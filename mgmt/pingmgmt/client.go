package pingmgmt

import (
	"errors"

	"ndn-dpdk/app/ping"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/core/nnduration"
)

type PingClientMgmt struct {
	App *ping.App
}

func (mg PingClientMgmt) getClient(index int) (client *pingclient.Client, e error) {
	if index >= len(mg.App.Tasks) {
		return nil, errors.New("Index out of range")
	}
	client = mg.App.Tasks[index].Client
	if client == nil {
		return nil, errors.New("Task has no Client")
	}
	return client, nil
}

func (mg PingClientMgmt) List(args struct{}, reply *[]int) error {
	var list []int
	for index, task := range mg.App.Tasks {
		if task.Client != nil {
			list = append(list, index)
		}
	}
	*reply = list
	return nil
}

func (mg PingClientMgmt) Start(args ClientStartArgs, reply *struct{}) error {
	client, e := mg.getClient(args.Index)
	if e != nil {
		return e
	}
	if client.Rx.IsRunning() || client.Tx.IsRunning() {
		return errors.New("Client is running")
	}

	if args.Interval != 0 {
		client.SetInterval(args.Interval.Duration())
	}
	if args.ClearCounters {
		client.ClearCounters()
	}
	client.Launch()
	return nil
}

func (mg PingClientMgmt) Stop(args ClientStopArgs, reply *struct{}) error {
	client, e := mg.getClient(args.Index)
	if e != nil {
		return e
	}

	client.Stop(args.RxDelay.Duration())
	return nil
}

func (mg PingClientMgmt) ReadCounters(args IndexArg, reply *pingclient.Counters) error {
	client, e := mg.getClient(args.Index)
	if e != nil {
		return e
	}

	*reply = client.ReadCounters()
	return nil
}

type IndexArg struct {
	Index int // Task index
}

type ClientStartArgs struct {
	IndexArg
	Interval      nnduration.Nanoseconds // Interest sending Interval
	ClearCounters bool                   // whether to clear counters
}

type ClientStopArgs struct {
	IndexArg
	RxDelay nnduration.Nanoseconds // sleep duration between stopping TX and stopping RX
}
