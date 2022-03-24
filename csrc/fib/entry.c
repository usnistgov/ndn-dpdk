#include "entry.h"
#include "../strategyapi/api.h"

static_assert(offsetof(SgCtx, global) == offsetof(FibSgInitCtx, global), "");
static_assert(offsetof(SgCtx, now) == offsetof(FibSgInitCtx, now), "");
static_assert(offsetof(SgCtx, fibEntry) == offsetof(FibSgInitCtx, entry), "");
static_assert(offsetof(SgCtx, fibEntryDyn) == offsetof(FibSgInitCtx, dyn), "");
static_assert(sizeof(SgCtx) <= offsetof(FibSgInitCtx, goHandle), "");
