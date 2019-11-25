package ping

/*
#include "input.h"
*/
import "C"
import (
	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Task struct {
	Face       iface.IFace
	Fetch      *fetch.Fetcher
	fetchStart bool
	Client     *pingclient.Client
	Server     *pingserver.Server
}

func newTask(face iface.IFace, cfg TaskConfig) (task Task, e error) {
	numaSocket := face.GetNumaSocket()
	task.Face = face
	if cfg.Fetch != nil {
		if task.Fetch, e = fetch.New(task.Face, cfg.Fetch.FetcherConfig); e != nil {
			return Task{}, e
		}
		task.Fetch.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, numaSocket))
		if cfg.Fetch.Name != nil {
			task.fetchStart = true
			task.Fetch.SetName(cfg.Fetch.Name)
			if cfg.Fetch.FinalSegNum != nil {
				task.Fetch.Logic.SetFinalSegNum(*cfg.Fetch.FinalSegNum)
			}
		}
	} else if cfg.Client != nil {
		if task.Client, e = pingclient.New(task.Face, *cfg.Client); e != nil {
			return Task{}, e
		}
		task.Client.SetLCores(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, numaSocket), dpdk.LCoreAlloc.Alloc(LCoreRole_ClientTx, numaSocket))
	}
	if cfg.Server != nil {
		if task.Server, e = pingserver.New(task.Face, *cfg.Server); e != nil {
			return Task{}, e
		}
		task.Server.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_Server, numaSocket))
	}
	return task, nil
}

func (task *Task) Launch() {
	if task.Server != nil {
		task.Server.Launch()
	}
	if task.Client != nil {
		task.Client.Launch()
	}
	if task.Fetch != nil && task.fetchStart {
		task.Fetch.Launch()
	}
}

func (task *Task) Close() error {
	if task.Server != nil {
		task.Server.Close()
	}
	if task.Client != nil {
		task.Client.Close()
	}
	if task.Fetch != nil {
		task.Fetch.Close()
	}
	task.Face.Close()
	return nil
}
