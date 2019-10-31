package pingmgmt

import (
	"errors"
	"time"

	"ndn-dpdk/app/ping"
	"ndn-dpdk/app/pingclient"
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
	if client.IsRunning() || client.Tx.IsRunning() {
		return errors.New("Client is running")
	}

	if args.Interval != 0 {
		client.SetInterval(args.Interval)
	}
	if args.ClearCounters {
		client.ClearCounters()
	}
	client.Launch()
	client.Tx.Launch()
	return nil
}

func (mg PingClientMgmt) Stop(args ClientStopArgs, reply *struct{}) error {
	client, e := mg.getClient(args.Index)
	if e != nil {
		return e
	}

	client.Tx.Stop()
	time.Sleep(args.RxDelay)
	client.Stop()
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
