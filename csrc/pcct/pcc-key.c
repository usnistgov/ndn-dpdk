#include "pcc-key.h"
#include "pcc-entry.h"

static_assert(sizeof(PccKeyExt) <= sizeof(PccEntry), "");

enum
{
  PccSearchDebugStringLength = 2 * NameHexBufferLength + 32,
};
static RTE_DEFINE_PER_LCORE(
  struct { char buffer[PccSearchDebugStringLength]; }, PccSearchDebugStringBuffer);

const char*
PccSearch_ToDebugString(const PccSearch* search)
{
  const char* buffer = RTE_PER_LCORE(PccSearchDebugStringBuffer).buffer;
  int pos = 0;
#define APPEND(...)                                                                                \
  do {                                                                                             \
    pos += snprintf(RTE_PTR_ADD(buffer, pos), PccSearchDebugStringLength - pos, __VA_ARGS__);      \
  } while (false)

  pos += LName_PrintHex(search->name, RTE_PTR_ADD(buffer, pos));

  APPEND(" ");
  if (search->fh.length == 0) {
    APPEND("(no-fh)");
  } else {
    pos += LName_PrintHex(search->fh, RTE_PTR_ADD(buffer, pos));
  }

#undef APPEND
  NDNDPDK_ASSERT(pos + 1 <= PccSearchDebugStringLength);
  return buffer;
}

bool
PccKey_MatchFieldWithExt_(LName name, const uint8_t* firstV, uint16_t firstCapacity,
                          const PccKeyExt* ext)
{
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
                          PccKeyExt* exts[])
{
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
