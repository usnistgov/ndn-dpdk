#include "pcc-key.h"
#include "pcc-entry.h"

static_assert(sizeof(PccKeyExt) <= sizeof(PccEntry), "");

const char*
PccSearch_ToDebugString(const PccSearch* search, char buffer[PccSearchDebugStringLength])
{
  int pos = 0;
#define append(...)                                                                                \
  do {                                                                                             \
    pos += snprintf(RTE_PTR_ADD(buffer, pos), PccSearchDebugStringLength - pos, __VA_ARGS__);      \
  } while (false)

  pos += LName_PrintHex(search->name, RTE_PTR_ADD(buffer, pos));

  append(" ");
  if (unlikely(search->fh.length == 0)) {
    append("(no-fh)");
  } else {
    pos += LName_PrintHex(search->fh, RTE_PTR_ADD(buffer, pos));
  }

#undef append
  return buffer;
}
