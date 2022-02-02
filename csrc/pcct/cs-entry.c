#include "cs-entry.h"

static const char* CsEntryKind_Strings[] = {
  [CsEntryNone] = "none",
  [CsEntryMemory] = "memory",
  [CsEntryDisk] = "disk",
  [CsEntryIndirect] = "indirect",
};

const char*
CsEntryKind_ToString(CsEntryKind kind)
{
  return CsEntryKind_Strings[kind];
}
