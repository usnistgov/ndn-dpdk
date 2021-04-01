package strategycode

/*
#include "../../csrc/strategycode/strategy-code.h"
*/
import "C"
import (
	"os"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"go4.org/must"
)

// External symbols available to eBPF programs, provided by ndn-dpdk/app/fwdp package.
var (
	Xsyms  unsafe.Pointer
	NXsyms int
)

func makeStrategyCode(name string, bpf *C.struct_rte_bpf) (sc *Strategy, e error) {
	if bpf == nil {
		return nil, eal.GetErrno()
	}

	var jit C.struct_rte_bpf_jit
	res := C.rte_bpf_get_jit(bpf, &jit)
	if res != 0 {
		C.rte_bpf_destroy(bpf)
		return nil, eal.Errno(-res)
	}

	tableLock.Lock()
	defer tableLock.Unlock()
	lastID++

	sc = (*Strategy)(eal.Zmalloc("Strategy", C.sizeof_StrategyCode, eal.NumaSocket{}))
	c := sc.ptr()
	c.id = C.int(lastID)
	c.name = C.CString(name)
	c.nRefs = 1
	c.bpf = bpf
	c.jit = jit._func
	table[lastID] = sc
	return sc, nil
}

var dotTextSection = C.CString(".text")

// Load loads a strategy BPF program from ELF object.
func Load(name string, elf []byte) (sc *Strategy, e error) {
	file, e := os.CreateTemp("", "strategy*.o")
	if e != nil {
		return nil, e
	}
	filename := file.Name()
	defer os.Remove(filename)
	if _, e := file.Write(elf); e != nil {
		return nil, e
	}
	must.Close(file)

	return LoadFile(name, filename)
}

// LoadFile loads a strategy BPF program from ELF file.
// If filename is empty, search for an ELF file in default locations.
func LoadFile(name, filename string) (sc *Strategy, e error) {
	if filename == "" {
		filename, e = bpf.Strategy.Find(name)
		if e != nil {
			return nil, e
		}
	}

	var prm C.struct_rte_bpf_prm
	prm.xsym = (*C.struct_rte_bpf_xsym)(Xsyms)
	prm.nb_xsym = (C.uint32_t)(NXsyms)
	prm.prog_arg._type = C.RTE_BPF_ARG_RAW

	filenameC := C.CString(filename)
	defer C.free(unsafe.Pointer(filenameC))
	bpf := C.rte_bpf_elf_load(&prm, filenameC, dotTextSection)
	return makeStrategyCode(name, bpf)
}

// MakeEmpty creates an empty BPF program.
// This is useful for unit testing.
func MakeEmpty(name string) *Strategy {
	var prm C.struct_rte_bpf_prm
	prm.ins = C.StrategyCode_GetEmptyProgram_(&prm.nb_ins)
	prm.prog_arg._type = C.RTE_BPF_ARG_RAW

	bpf := C.rte_bpf_load(&prm)
	sc, e := makeStrategyCode(name, bpf)
	if e != nil {
		panic(e)
	}
	return sc
}
