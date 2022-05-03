#include "cs-entry.h"

const char* CsEntryKind_Strings_[] = {
  [CsEntryNone] = "none",
  [CsEntryMemory] = "memory",
  [CsEntryDisk] = "disk",
  [CsEntryIndirect] = "indirect",
};
