#include "pit-dn-up-it.h"
#include "pit.h"

bool
__PitDnUpIt_Extend(__PitDnUpIt* it, Pit* pit, int maxInExt, size_t offsetInExt)
{
  assert(it->i == it->max);
  assert(*it->nextPtr == NULL);

  // allocate PitEntryExt
  struct rte_mempool* mp = Pcct_ToMempool(Pit_ToPcct(pit));
  PitEntryExt* ext;
  int res = rte_mempool_get(mp, (void**)&ext);
  if (unlikely(res != 0)) {
    return false;
  }

  // clear PitEntryExt
  ext->dns[0].face = FACEID_INVALID;
  ext->ups[0].face = FACEID_INVALID;
  ext->next = NULL;

  // chain after PitEntry or existing PitEntryExt
  *it->nextPtr = ext;

  // update iterator
  it->i = 0;
  it->max = maxInExt;
  it->array = RTE_PTR_ADD(ext, offsetInExt);
  it->nextPtr = &ext->next;
  return true;
}
