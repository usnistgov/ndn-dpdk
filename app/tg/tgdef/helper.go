package tgdef

import (
	"reflect"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"go.uber.org/multierr"
)

// GatherWorkers helps implementing Module.Workers.
func GatherWorkers(list any) (workers []ealthread.ThreadWithRole) {
	val := reflect.ValueOf(list)
	if val.Kind() != reflect.Slice {
		panic("list is not a slice")
	}
	for i, count := 0, val.Len(); i < count; i++ {
		workers = append(workers, val.Index(i).Interface().(ealthread.ThreadWithRole))
	}
	return workers
}

// LaunchWorkers helps implementing Module.Launch.
func LaunchWorkers(workers []ealthread.ThreadWithRole) {
	for _, w := range workers {
		ealthread.Launch(w)
	}
}

// StopWorkers helps implementing Module.Stop.
func StopWorkers(workers []ealthread.ThreadWithRole) error {
	errs := []error{}
	for _, w := range workers {
		errs = append(errs, w.Stop())
	}
	return multierr.Combine(errs...)
}
