// Package fileserver implements a file server.
package fileserver

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

var logger = logging.New("fileserver")

// Server represents a file server.
type Server struct {
	mounts          []Mount
	workers         []*worker
	VersionBypassHi uint32
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

// Counters retrieves counters.
func (p *Server) Counters() (cnt Counters) {
	for _, w := range p.workers {
		w.addToCounters(&cnt)
	}
	return cnt
}

// Launch launches all workers.
func (p *Server) Launch() {
	tgdef.LaunchWorkers(p.workers)
}

// Stop stops all workers.
func (p *Server) Stop() error {
	return tgdef.StopWorkers(p.workers)
}

// Close closes the server.
func (p *Server) Close() error {
	errs := []error{p.Stop()}
	for _, w := range p.workers {
		errs = append(errs, w.close())
	}
	p.workers = nil
	for _, m := range p.mounts {
		errs = append(errs, m.closeDirectory())
	}
	p.mounts = nil
	return errors.Join(errs...)
}

// New creates a Server.
func New(face iface.Face, cfg Config) (p *Server, e error) {
	if e := cfg.Validate(); e != nil {
		return nil, e
	}

	faceID, socket := face.ID(), face.NumaSocket()

	p = &Server{
		VersionBypassHi: cfg.versionBypassHi,
	}

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
