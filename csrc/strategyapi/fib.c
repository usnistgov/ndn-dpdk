#include "fib.h"
#include "../fib/entry.h"

static_assert(offsetof(SgFibEntry, nNexthops) == offsetof(FibEntry, nNexthops), "");
static_assert(offsetof(SgFibEntry, nexthops) == offsetof(FibEntry, nexthops), "");
static_assert(offsetof(SgFibEntry, scratch) == offsetof(SgFibEntry, scratch), "");
static_assert(sizeof(SgFibEntry) == sizeof(FibEntry), "");

static_assert(sizeof(SgFibNexthopFilter) == sizeof(FibNexthopFilter), "");

static_assert(SG_FIB_ENTRY_MAX_NEXTHOPS == FibMaxNexthops, "");
static_assert(SG_FIB_ENTRY_SCRATCH == FibScratchSize, "");
