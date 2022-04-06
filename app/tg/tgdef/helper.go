package tgdef

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"go.uber.org/multierr"
)

// GatherWorkers helps implementing Module.Workers.
func GatherWorkers[W ealthread.ThreadWithRole](list []W) (workers []ealthread.ThreadWithRole) {
	for _, w := range list {
		workers = append(workers, w)
	}
	return workers
}

// LaunchWorkers helps implementing Module.Launch.
func LaunchWorkers[W ealthread.Thread](workers []W) {
	for _, w := range workers {
		ealthread.Launch(w)
	}
}

// StopWorkers helps implementing Module.Stop.
func StopWorkers[W ealthread.Thread](workers []W) error {
	errs := []error{}
	for _, w := range workers {
		errs = append(errs, w.Stop())
	}
	return multierr.Combine(errs...)
}
