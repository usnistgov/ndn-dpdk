package ping

import (
	"ndn-dpdk/app/fetch"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type Task struct {
	Face   iface.IFace
	Server *pingserver.Server
	Client *pingclient.Client
	Fetch  []*fetch.Fetcher
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
	} else if cfg.Fetch > 0 {
		for fetchId := 0; fetchId < cfg.Fetch; fetchId++ {
			fetcher, e := fetch.New(fetchId, task.Face, cfg.FetchCfg)
			if e != nil {
				return Task{}, e
			}
			fetcher.SetLCore(dpdk.LCoreAlloc.Alloc(LCoreRole_ClientRx, socket))
			task.Fetch = append(task.Fetch, fetcher)
		}
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
}

func (task *Task) Close() error {
	if task.Server != nil {
		task.Server.Close()
	}
	if task.Client != nil {
		task.Client.Close()
	}
	for _, fetcher := range task.Fetch {
		fetcher.Close()
	}
	task.Face.Close()
	return nil
}
