package strategycode

/*
#include "../../csrc/strategycode/strategy-code.h"
#include "../../csrc/strategycode/sec.h"

extern void go_StrategyCode_Free(uintptr_t goHandle);
*/
import "C"
import (
	"debug/elf"
	"errors"
	"fmt"
	"os"
	"runtime/cgo"
	"strings"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/bpf"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
)

// Xsyms references eBPF external symbols.
type Xsyms struct {
	ptr *C.struct_rte_bpf_xsym
	n   C.uint32_t
}

// Assign sets eBPF external symbols.
func (x *Xsyms) Assign(ptr unsafe.Pointer, n int) {
	x.ptr, x.n = (*C.struct_rte_bpf_xsym)(ptr), C.uint32_t(n)
}

// External symbols available to eBPF programs.
var (
	XsymsMain Xsyms // fwdp init
	XsymsInit Xsyms
)

// Strategy is a reference of a forwarding strategy.
type Strategy struct {
	c      *C.StrategyCode
	id     int
	name   string
	init   C.StrategyCodeProg
	schema *gojsonschema.Schema
}

// Ptr returns *C.Strategy pointer.
func (sc *Strategy) Ptr() unsafe.Pointer {
	return unsafe.Pointer(sc.c)
}

// ID returns numeric ID.
func (sc *Strategy) ID() int {
	return sc.id
}

// Name returns short name.
func (sc *Strategy) Name() string {
	return sc.name
}

// ValidateParams validates JSON parameters.
func (sc *Strategy) ValidateParams(params map[string]any) error {
	if sc.schema == nil {
		if len(params) != 0 {
			return errors.New("strategy does not accept parameters")
		}
		return nil
	}
	result, e := sc.schema.Validate(gojsonschema.NewGoLoader(params))
	switch {
	case e != nil:
		return e
	case result.Valid():
		return nil
	default:
		var b strings.Builder
		fmt.Fprintln(&b, "strategy parameters failed schema validation:")
		for _, desc := range result.Errors() {
			fmt.Fprintln(&b, "-", desc)
		}
		return errors.New(b.String())
	}
}

// InitFunc returns the init function, or nil if it does not exist.
func (sc *Strategy) InitFunc() func(arg unsafe.Pointer, sizeofArg uintptr) uint64 {
	if sc.init.jit == nil {
		return nil
	}
	return func(arg unsafe.Pointer, sizeofArg uintptr) uint64 {
		return uint64(C.StrategyCodeProg_Run(sc.init, arg, C.size_t(sizeofArg)))
	}
}

// CountRefs returns number of references.
// Each FIB entry using the strategy has a reference.
// There's also a reference from table.go.
func (sc *Strategy) CountRefs() int {
	return int(sc.c.nRefs)
}

// Unref reduces the number of references by one.
// The strategy cannot be retrieved via Get(), Find(), List().
// It will be unloaded when its reference count reaches zero.
func (sc *Strategy) Unref() {
	tableLock.Lock()
	defer tableLock.Unlock()
	delete(table, sc.ID())
	C.StrategyCode_Unref(sc.c)
}

func (sc *Strategy) free() {
	if sc.c.goHandle != 0 {
		cgo.Handle(sc.c.goHandle).Delete()
	}
	freeProg(sc.c.main)
	freeProg(sc.init)
	eal.Free(sc.c)
}

func (sc *Strategy) String() string {
	if sc == nil {
		return "0@nil"
	}
	return fmt.Sprintf("%d@%p", sc.ID(), sc.c)
}

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
	file.Close()

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

	elfFile, e := elf.Open(filename)
	if e != nil {
		return nil, e
	}
	defer elfFile.Close()

	tableLock.Lock()
	defer tableLock.Unlock()
	lastID++

	sc = &Strategy{
		c:    (*C.StrategyCode)(eal.Zmalloc("Strategy", C.sizeof_StrategyCode, eal.NumaSocket{})),
		id:   lastID,
		name: name,
	}
	defer func(sc *Strategy) {
		if e != nil {
			sc.free()
		}
	}(sc)

	if sec := elfFile.Section(C.SGSEC_MAIN); sec != nil {
		if sc.c.main, e = makeProg(filename, C.SGSEC_MAIN, XsymsMain); e != nil {
			return nil, fmt.Errorf("jit-compile %s: %w", C.SGSEC_MAIN, e)
		}
	} else {
		return nil, fmt.Errorf("missing %s section", C.SGSEC_MAIN)
	}

	if sec := elfFile.Section(C.SGSEC_INIT); sec != nil {
		if sc.init, e = makeProg(filename, C.SGSEC_INIT, XsymsInit); e != nil {
			return nil, fmt.Errorf("jit-compile %s: %w", C.SGSEC_INIT, e)
		}
	}

	if sec := elfFile.Section(C.SGSEC_SCHEMA); sec != nil {
		text, e := sec.Data()
		if e != nil {
			return nil, fmt.Errorf("read %s: %w", C.SGSEC_SCHEMA, e)
		}
		sc.schema, e = gojsonschema.NewSchema(gojsonschema.NewBytesLoader(text))
		if e != nil {
			return nil, fmt.Errorf("load %s: %w", C.SGSEC_SCHEMA, e)
		}
	}

	table[sc.id] = sc
	sc.c.id = C.int(sc.id)
	sc.c.goHandle = C.uintptr_t(cgo.NewHandle(sc))
	C.StrategyCode_Ref(sc.c)
	return sc, nil
}

// MakeEmpty returns an empty strategy for unit testing.
// Panics on error.
func MakeEmpty(name string) *Strategy {
	filename, e := bpf.Strategy.Find("empty")
	if e != nil {
		logger.Panic("MakePanic error", zap.Error(e))
	}

	sc, e := LoadFile(name, filename)
	if e != nil {
		logger.Panic("MakePanic error", zap.Error(e))
	}

	return sc
}

func makeProg(filename, section string, xsyms Xsyms) (prog C.StrategyCodeProg, e error) {
	filenameC, sectionC := C.CString(filename), C.CString(section)
	defer func() {
		C.free(unsafe.Pointer(filenameC))
		C.free(unsafe.Pointer(sectionC))
	}()

	var prm C.struct_rte_bpf_prm
	prm.xsym, prm.nb_xsym = xsyms.ptr, xsyms.n
	prm.prog_arg._type = C.RTE_BPF_ARG_RAW

	prog.bpf = C.rte_bpf_elf_load(&prm, filenameC, sectionC)
	if prog.bpf == nil {
		return C.StrategyCodeProg{}, eal.GetErrno()
	}

	var jit C.struct_rte_bpf_jit
	if res := C.rte_bpf_get_jit(prog.bpf, &jit); res != 0 {
		C.rte_bpf_destroy(prog.bpf)
		return C.StrategyCodeProg{}, eal.MakeErrno(res)
	}
	prog.jit = jit._func
	return prog, nil
}

func freeProg(prog C.StrategyCodeProg) {
	if prog.bpf != nil {
		C.rte_bpf_destroy(prog.bpf)
	}
}

//export go_StrategyCode_Free
func go_StrategyCode_Free(goHandle C.uintptr_t) {
	sc := cgo.Handle(goHandle).Value().(*Strategy)
	sc.free()
}

func init() {
	C.StrategyCode_Free = C.StrategyCode_FreeFunc(C.go_StrategyCode_Free)

	XsymsInit.ptr = C.SgInitGetXsyms(&XsymsInit.n)
}
