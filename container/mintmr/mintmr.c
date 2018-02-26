#include "mintmr.h"

MinSched*
MinSched_New(int nSlotBits, TscDuration interval, MinTmrCallback cb,
             void* cbarg)
{
  uint16_t nSlots = 1 << nSlotBits;
  MinSched* sched =
    rte_zmalloc("MinSched", sizeof(MinSched) + nSlots * sizeof(MinTmr), 0);
  sched->interval = interval;
  sched->cb = cb;
  sched->cbarg = cbarg;
  sched->nSlots = nSlots;
  sched->slotMask = nSlots - 1;
  sched->lastSlot = nSlots - 1;
  sched->nextTime = rte_get_tsc_cycles();

  for (uint16_t i = 0; i < nSlots; ++i) {
    MinTmr* slot = &sched->slot[i];
    slot->next = slot->prev = slot;
  }
  return sched;
}

void
MinSched_Close(MinSched* sched)
{
  rte_free(sched);
}

void
__MinSched_Trigger(MinSched* sched, TscTime now)
{
  while (sched->nextTime <= now) {
    sched->lastSlot = (sched->lastSlot + 1) & sched->slotMask;
    sched->nextTime += sched->interval;
    MinTmr* slot = &sched->slot[sched->lastSlot];

    MinTmr* next;
    for (MinTmr* tmr = slot->next; tmr != slot; tmr = next) {
      next = tmr->next;
      (sched->cb)(tmr, sched->cbarg);
      MinTmr_Init(tmr);
    }
    slot->next = slot->prev = slot;
  }
}

bool
MinTmr_After(MinTmr* tmr, TscDuration after, MinSched* sched)
{
  MinSched_Trigger(sched);

  uint64_t nSlotsAway = after / sched->interval + 1;
  if (unlikely(nSlotsAway >= sched->nSlots)) {
    return false;
  }

  MinTmr* slot = &sched->slot[(sched->lastSlot + nSlotsAway) & sched->slotMask];
  tmr->next = slot->next;
  tmr->next->prev = tmr;
  slot->next = tmr;
  tmr->prev = slot;
  return true;
}
