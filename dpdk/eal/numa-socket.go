package eal

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"encoding/json"
	"reflect"
	"strconv"
)

// NumaSocket represents a NUMA socket.
// Zero value is SOCKET_ID_ANY.
type NumaSocket struct {
	v int // socket ID + 1
}

// NumaSocketFromID converts socket ID to NumaSocket.
func NumaSocketFromID(id int) (socket NumaSocket) {
	if id < 0 || id > C.RTE_MAX_NUMA_NODES {
		return socket
	}
	socket.v = id + 1
	return socket
}

// ID returns NUMA socket ID.
func (socket NumaSocket) ID() int {
	return socket.v - 1
}

// IsAny returns true if this represents SOCKET_ID_ANY.
func (socket NumaSocket) IsAny() bool {
	return socket.v == 0
}

// Match returns true if either NumaSocket is SOCKET_ID_ANY, or both are the same NumaSocket.
func (socket NumaSocket) Match(other NumaSocket) bool {
	return socket.IsAny() || other.IsAny() || socket.v == other.v
}

func (socket NumaSocket) String() string {
	if socket.IsAny() {
		return "any"
	}
	return strconv.Itoa(socket.ID())
}

// MarshalJSON encodes NUMA socket as number.
// Any is encoded as null.
func (socket NumaSocket) MarshalJSON() ([]byte, error) {
	if socket.IsAny() {
		return json.Marshal(nil)
	}
	return json.Marshal(socket.ID())
}

// WithNumaSocket interface is implemented by types that have an associated or preferred NUMA socket.
type WithNumaSocket interface {
	NumaSocket() NumaSocket
}

// NumaSocketsOf collects associated/preferred NUMA sockets of a list of objects.
// list must be a slice of objects that implement WithNumaSocket; panics otherwise.
func NumaSocketsOf(list interface{}) (result []NumaSocket) {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		panic(v.Type().String() + " is not a slice")
	}

	result = make([]NumaSocket, v.Len())
	for i := range result {
		result[i] = v.Index(i).Interface().(WithNumaSocket).NumaSocket()
	}
	return result
}
