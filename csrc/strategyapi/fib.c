#include "fib.h"
#include "../fib/nexthop-filter.h"

static_assert(sizeof(SgFibEntryDyn) == sizeof(FibEntryDyn), "");
static_assert(offsetof(SgFibEntryDyn, scratch) == offsetof(FibEntryDyn, scratch), "");
static_assert(offsetof(SgFibEntryDyn, rtt) == offsetof(FibEntryDyn, rtt), "");

static_assert(sizeof(SgFibEntry) <= sizeof(FibEntry), "");
static_assert(offsetof(SgFibEntry, nNexthops) == offsetof(FibEntry, nNexthops), "");
static_assert(offsetof(SgFibEntry, nexthops) == offsetof(FibEntry, nexthops), "");

static_assert(sizeof(SgFibNexthopFilter) == sizeof(FibNexthopFilter), "");
