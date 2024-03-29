#include "pcc-entry.h"
#include "pcct.h"

static_assert(offsetof(PccSlot, pccEntry) == offsetof(PitEntry, pccEntry), "");
static_assert(offsetof(PccSlot, pccEntry) == offsetof(CsEntry, pccEntry), "");

PccSlotIndex
PccEntry_AllocateSlot_(PccEntry* entry, PccSlot** slot) {
#define AssignSlot(s)                                                                              \
  do {                                                                                             \
    POISON(&s);                                                                                    \
    (s).pccEntry = entry;                                                                          \
    *slot = &(s);                                                                                  \
  } while (false)

  if (entry->slot1.pccEntry == NULL) {
    AssignSlot(entry->slot1);
    return PCC_SLOT1;
  }

  if (entry->ext == NULL) {
    int res = rte_mempool_get(rte_mempool_from_obj(entry), (void**)&entry->ext);
    if (unlikely(res != 0)) {
      goto FAIL;
    }
    entry->ext->slot3.pccEntry = NULL;

    AssignSlot(entry->ext->slot2);
    return PCC_SLOT2;
  }

  if (entry->ext->slot2.pccEntry == NULL) {
    AssignSlot(entry->ext->slot2);
    return PCC_SLOT2;
  }
  if (entry->ext->slot3.pccEntry == NULL) {
    AssignSlot(entry->ext->slot3);
    return PCC_SLOT3;
  }

FAIL:
  *slot = NULL;
  return PCC_SLOT_NONE;

#undef AssignSlot
}

PccSlotIndex
PccEntry_ClearSlot_(PccEntry* entry, PccSlotIndex slot) {
  switch (slot) {
    case PCC_SLOT_NONE:
      goto FINISH;
    case PCC_SLOT1: {
      POISON(&entry->slot1);
      entry->slot1.pccEntry = NULL;
      goto FINISH;
    }
    case PCC_SLOT2: {
      POISON(&entry->ext->slot2);
      entry->ext->slot2.pccEntry = NULL;
      if (entry->ext->slot3.pccEntry != NULL) {
        goto FINISH;
      }
      break;
    }
    case PCC_SLOT3: {
      POISON(&entry->ext->slot3);
      entry->ext->slot3.pccEntry = NULL;
      if (entry->ext->slot2.pccEntry != NULL) {
        goto FINISH;
      }
      break;
    }
  }

  rte_mempool_put(rte_mempool_from_obj(entry), entry->ext);
  entry->ext = NULL;

FINISH:
  return PCC_SLOT_NONE;
}
