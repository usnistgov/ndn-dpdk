#include "api-pit.h"
#include "../container/pcct/pit-entry.h"

static_assert(offsetof(SgPitDn, expiry) == offsetof(PitDn, expiry), "");
static_assert(offsetof(SgPitDn, face) == offsetof(PitDn, face), "");

static_assert(offsetof(SgPitUp, lastTx) == offsetof(PitUp, lastTx), "");
static_assert(offsetof(SgPitUp, face) == offsetof(PitUp, face), "");
static_assert(offsetof(SgPitUp, nack) == offsetof(PitUp, nack), "");

static_assert(offsetof(SgPitEntry, ext) == offsetof(PitEntry, ext), "");
static_assert(offsetof(SgPitEntry, dns) == offsetof(PitEntry, dns), "");
static_assert(offsetof(SgPitEntry, ups) == offsetof(PitEntry, ups), "");
static_assert(sizeof(SgPitEntry) == sizeof(PitEntry), "");

static_assert(offsetof(SgPitEntryExt, next) == offsetof(PitEntryExt, next), "");
static_assert(offsetof(SgPitEntryExt, dns) == offsetof(PitEntryExt, dns), "");
static_assert(offsetof(SgPitEntryExt, ups) == offsetof(PitEntryExt, ups), "");
static_assert(sizeof(SgPitEntryExt) == sizeof(PitEntryExt), "");

static_assert(SG_PIT_ENTRY_MAX_DNS == PIT_ENTRY_MAX_DNS, "");
static_assert(SG_PIT_ENTRY_MAX_UPS == PIT_ENTRY_MAX_UPS, "");
static_assert(SG_PIT_ENTRY_EXT_MAX_DNS == PIT_ENTRY_EXT_MAX_DNS, "");
static_assert(SG_PIT_ENTRY_EXT_MAX_UPS == PIT_ENTRY_EXT_MAX_UPS, "");
