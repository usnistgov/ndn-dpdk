package mempool

/*
#include "../../csrc/core/common.h"
#include <rte_memory.h>
#include <rte_mempool.h>
#include <rte_memzone.h>
*/
import "C"
import (
	"errors"
	"io"
	"os"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlMemoryDiagType *graphql.Object
)

func makeFileDumpResolver(f func(fp *C.FILE)) graphql.FieldResolveFn {
	return func(p graphql.ResolveParams) (interface{}, error) {
		file, e := os.CreateTemp("", "")
		if e != nil {
			return nil, e
		}
		filename := file.Name()
		defer os.Remove(filename)

		conn, e := file.SyscallConn()
		if e != nil {
			return nil, e
		}
		conn.Write(func(fd uintptr) bool {
			mode := []C.char{'w', 0}
			fp := C.fdopen(C.int(fd), &mode[0])
			f(fp)
			C.fflush(fp)
			return true
		})

		file.Seek(0, 0)
		content, e := io.ReadAll(file)
		if e != nil {
			return nil, e
		}
		return string(content), e
	}
}

func init() {
	GqlMemoryDiagType = graphql.NewObject(graphql.ObjectConfig{
		Name: "MemoryDiag",
		Fields: graphql.Fields{
			"physmemLayout": &graphql.Field{
				Description: "Physical memory layout.",
				Type:        gqlserver.NonNullString,
				Resolve:     makeFileDumpResolver(func(fp *C.FILE) { C.rte_dump_physmem_layout(fp) }),
			},
			"memzones": &graphql.Field{
				Description: "Reserved memzones.",
				Type:        gqlserver.NonNullString,
				Resolve:     makeFileDumpResolver(func(fp *C.FILE) { C.rte_memzone_dump(fp) }),
			},
			"mempoolList": &graphql.Field{
				Description: "Status of mempools.",
				Type:        gqlserver.NonNullString,
				Resolve:     makeFileDumpResolver(func(fp *C.FILE) { C.rte_mempool_list_dump(fp) }),
			},
			"malloc": &graphql.Field{
				Description: "malloc statistics.",
				Type:        gqlserver.NonNullString,
				Resolve:     makeFileDumpResolver(func(fp *C.FILE) { C.rte_malloc_dump_stats(fp, nil) }),
			},
			"heap": &graphql.Field{
				Description: "Contents of malloc heaps.",
				Type:        gqlserver.NonNullString,
				Resolve:     makeFileDumpResolver(func(fp *C.FILE) { C.rte_malloc_dump_heaps(fp) }),
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "memoryDiag",
		Description: "DPDK memory-related diagnose reports.",
		Type:        graphql.NewNonNull(GqlMemoryDiagType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if eal.MainThread == nil {
				return nil, errors.New("EAL not ready")
			}
			return "", nil
		},
	})
}
