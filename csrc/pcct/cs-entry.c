#include "cs-entry.h"

const char* CsEntryKindString[] = {
  [CsEntryNone] = "none",
  [CsEntryMemory] = "memory",
  [CsEntryDisk] = "disk",
  [CsEntryIndirect] = "indirect",
};
