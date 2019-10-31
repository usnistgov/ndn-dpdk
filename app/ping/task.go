package ping

/*
#include "input.h"
*/
import "C"
import (
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Task struct {
	Face   iface.IFace
	Client *pingclient.Client
	Server *pingserver.Server
}

func newTask(face iface.IFace, cfg TaskConfig) (task Task, e error) {
	numaSocket := face.GetNumaSocket()
	task.Face = face
	if cfg.Client != nil {
		if task.Client, e = pingclient.New(task.Face, *cfg.Client); e != nil {
			return Task{}, e
		}
		task.Client.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, numaSocket))
		task.Client.Tx.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientTx, numaSocket))
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
		task.Client.Tx.Launch()
	}
}

func (task *Task) Close() error {
	if task.Server != nil {
		task.Server.Close()
	}
	if task.Client != nil {
		task.Client.Close()
	}
	task.Face.Close()
	return nil
}
