package ping

import (
	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/inputdemux"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Task struct {
	Face   iface.IFace
	Server *pingserver.Server
	Client *pingclient.Client
	Fetch  *fetch.Fetcher
}

func newTask(face iface.IFace, cfg TaskConfig) (task Task, e error) {
	socket := face.GetNumaSocket()
	task.Face = face

	if cfg.Server != nil {
		if task.Server, e = pingserver.New(task.Face, *cfg.Server); e != nil {
			return Task{}, e
		}
		task.Server.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_Server, socket))
	}

	if cfg.Client != nil {
		if task.Client, e = pingclient.New(task.Face, *cfg.Client); e != nil {
			return Task{}, e
		}
		task.Client.SetLCores(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, socket), dpdk.LCoreAlloc.Alloc(LCoreRole_ClientTx, socket))
	} else if cfg.Fetch != nil {
		if task.Fetch, e = fetch.New(task.Face, *cfg.Fetch); e != nil {
			return Task{}, e
		}
		for i, last := 0, task.Fetch.CountThreads(); i < last; i++ {
			task.Fetch.GetThread(i).SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, socket))
		}
	}

	return task, nil
}

func (task *Task) ConfigureDemux(demux3 inputdemux.Demux3) {
	demuxI := demux3.GetInterestDemux()
	demuxD := demux3.GetDataDemux()
	demuxN := demux3.GetNackDemux()

	if task.Server != nil {
		demuxI.InitFirst()
		demuxI.SetDest(0, task.Server.GetRxQueue())
	}

	if task.Client != nil {
		demuxD.InitFirst()
		demuxN.InitFirst()
		q := task.Client.GetRxQueue()
		demuxD.SetDest(0, q)
		demuxN.SetDest(0, q)
	} else if task.Fetch != nil {
		demuxD.InitToken()
		demuxN.InitToken()
		for i, last := 0, task.Fetch.CountProcs(); i < last; i++ {
			q := task.Fetch.GetRxQueue(i)
			demuxD.SetDest(i, q)
			demuxN.SetDest(i, q)
		}
	}
}

func (task *Task) Launch() {
	if task.Server != nil {
		task.Server.Launch()
	}
	if task.Client != nil {
		task.Client.Launch()
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
