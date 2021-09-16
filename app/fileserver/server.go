package fileserver

import (
	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

var logger = logging.New("fileserver")

// Server represents a file server.
type Server struct {
	workers []*worker
	mounts  []Mount
}

var _ tgdef.Producer = &Server{}

// Mounts returns mount entries.
func (p Server) Mounts() []Mount {
	return p.mounts
}

// Face returns the associated face.
func (p *Server) Face() iface.Face {
	return p.workers[0].face()
}

// ConnectRxQueues connects Interest InputDemux to RxQueues.
func (p *Server) ConnectRxQueues(demuxI *iface.InputDemux) {
	demuxI.InitGenericHash(len(p.workers))
	for i, w := range p.workers {
		demuxI.SetDest(i, w.rxQueue())
	}
}

// Workers returns worker threads.
func (p *Server) Workers() []ealthread.ThreadWithRole {
	return tgdef.GatherWorkers(p.workers)
}

// Launch launches all workers.
func (p *Server) Launch() {
	tgdef.LaunchWorkers(p.Workers())
}

// Stop stops all workers.
func (p *Server) Stop() error {
	return tgdef.StopWorkers(p.Workers())
}

// Close closes the server.
func (p *Server) Close() error {
	errs := []error{p.Stop()}
	for _, w := range p.workers {
		errs = append(errs, w.close())
	}
	p.workers = nil
	for _, m := range p.mounts {
		m.closeDirectory()
	}
	p.mounts = nil
	return nil
}

// New creates a Server.
func New(face iface.Face, cfg Config) (p *Server, e error) {
	if e := cfg.Validate(); e != nil {
		return nil, e
	}
	if e := cfg.checkPayloadMempool(cfg.SegmentLen); e != nil {
		return nil, e
	}

	faceID := face.ID()
	socket := face.NumaSocket()

	p = &Server{}
	for _, m := range cfg.Mounts {
		if e := m.openDirectory(); e != nil {
			must.Close(p)
			return nil, e
		}
		p.mounts = append(p.mounts, m)
	}
	copy(cfg.Mounts, p.mounts)

	for i := 0; i < cfg.NThreads; i++ {
		w, e := newWorker(faceID, socket, cfg)
		if e != nil {
			must.Close(p)
			return nil, e
		}
		p.workers = append(p.workers, w)
	}
	return p, nil
}
