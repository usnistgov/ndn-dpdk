#include "pit-iterator.h"
#include "pit.h"

bool
PitDnUpIt_Extend_(PitDnUpIt_* it, int maxInExt, size_t offsetInExt) {
  NDNDPDK_ASSERT(it->i == it->max);
  NDNDPDK_ASSERT(*it->nextPtr == NULL);

  // allocate PitEntryExt
  PitEntryExt* ext;
  int res = rte_mempool_get(rte_mempool_from_obj(it->pccEntry), (void**)&ext);
  if (unlikely(res != 0)) {
    return false;
  }
  POISON(ext);

  // clear PitEntryExt
  ext->dns[0].face = 0;
  ext->ups[0].face = 0;
  ext->next = NULL;

  // chain after PitEntry or existing PitEntryExt
  *it->nextPtr = ext;

  // update iterator
  PitDnUpIt_EnterExt_(it, ext, maxInExt, offsetInExt);
  return true;
}
