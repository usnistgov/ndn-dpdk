#include "pcc-key.h"
#include "../core/base16.h"
#include "../core/logger.h"
#include "pcc-entry.h"

static_assert(sizeof(PccKeyExt) <= sizeof(PccEntry), "");

const char*
PccSearch_ToDebugString(const PccSearch* search) {
  DebugString_Use(2 * Base16_BufferSize(NameMaxLength) + 32);

  DebugString_Append(Base16_Encode, search->name.value, search->name.length);
  DebugString_Append(snprintf, ",");

  if (search->fh.length == 0) {
    DebugString_Append(snprintf, "(no-fh)");
  } else {
    DebugString_Append(Base16_Encode, search->fh.value, search->fh.length);
  }

  DebugString_Return();
}

bool
PccKey_MatchFieldWithExt_(LName name, const uint8_t* firstV, uint16_t firstCapacity,
                          const PccKeyExt* ext) {
  NDNDPDK_ASSERT(name.length > firstCapacity);
  if (memcmp(firstV, name.value, firstCapacity) != 0) {
    return false;
  }
  for (uint16_t offset = firstCapacity; offset < name.length; offset += PccKeyExtCapacity) {
    NDNDPDK_ASSERT(ext != NULL);
    if (memcmp(ext->value, RTE_PTR_ADD(name.value, offset),
               RTE_MIN(name.length - offset, PccKeyExtCapacity)) != 0) {
      return false;
    }
    ext = ext->next;
  }
  return true;
}

int
PccKey_WriteFieldWithExt_(LName name, uint8_t* firstV, uint16_t firstCapacity, PccKeyExt** next,
                          PccKeyExt* exts[]) {
  NDNDPDK_ASSERT(name.length > firstCapacity);
  rte_memcpy(firstV, name.value, firstCapacity);
  int nExts = 0;
  for (uint16_t offset = firstCapacity; offset < name.length; offset += PccKeyExtCapacity) {
    PccKeyExt* ext = exts[nExts++];
    *next = ext;
    rte_memcpy(ext->value, RTE_PTR_ADD(name.value, offset),
               RTE_MIN(name.length - offset, PccKeyExtCapacity));
    next = &ext->next;
  }
  *next = NULL;
  return nExts;
}
