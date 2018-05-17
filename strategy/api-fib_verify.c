#include "api-fib.h"
#include "../container/fib/entry.h"

static_assert(offsetof(SgFibEntryDyn, scratch) ==
                offsetof(FibEntryDyn, scratch),
              "");
static_assert(sizeof(SgFibEntryDyn) == sizeof(FibEntryDyn), "");

static_assert(offsetof(SgFibEntry, dyn) == offsetof(FibEntry, dyn), "");
static_assert(offsetof(SgFibEntry, nNexthops) == offsetof(FibEntry, nNexthops),
              "");
static_assert(offsetof(SgFibEntry, nexthops) == offsetof(FibEntry, nexthops),
              "");
static_assert(sizeof(SgFibEntry) == sizeof(FibEntry), "");

static_assert(sizeof(SgFibNexthopFilter) == sizeof(FibNexthopFilter), "");

static_assert(SG_FIB_ENTRY_MAX_NEXTHOPS == FIB_ENTRY_MAX_NEXTHOPS, "");
static_assert(SG_FIB_DYN_SCRATCH == FIB_DYN_SCRATCH, "");
