#include "mintmr.h"

#include "../../core/logger.h"

INIT_ZF_LOG(MinTmr);

MinSched*
MinSched_New(int nSlotBits,
             TscDuration interval,
             MinTmrCallback cb,
             void* cbarg)
{
  uint16_t nSlots = 1 << nSlotBits;
  assert(nSlots != 0);

  MinSched* sched =
    rte_zmalloc("MinSched", sizeof(MinSched) + nSlots * sizeof(MinTmr), 0);
  sched->interval = interval;
  sched->cb = cb;
  sched->cbarg = cbarg;
  sched->nSlots = nSlots;
  sched->slotMask = nSlots - 1;
  sched->lastSlot = nSlots - 1;
  sched->nextTime = rte_get_tsc_cycles();

  ZF_LOGI("%p New(slots=%" PRIu16 " interval=%" PRIu64 " cb=%p)",
          sched,
          sched->nSlots,
          sched->interval,
          cb);

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
MinSched_Trigger_(MinSched* sched, TscTime now)
{
  while (sched->nextTime <= now) {
    sched->lastSlot = (sched->lastSlot + 1) & sched->slotMask;
    MinTmr* slot = &sched->slot[sched->lastSlot];
    ZF_LOGV("%p Trigger() slot=%" PRIu16 " time=%" PRIu64 " now=%" PRIu64,
            sched,
            sched->lastSlot,
            sched->nextTime,
            now);
    sched->nextTime += sched->interval;

    MinTmr* next;
    for (MinTmr* tmr = slot->next; tmr != slot; tmr = next) {
      next = tmr->next;
      ZF_LOGD(
        "%p Trigger() slot=%" PRIu16 " tmr=%p", sched, sched->lastSlot, tmr);
      ++sched->nTriggered;
      (sched->cb)(tmr, sched->cbarg);
      MinTmr_Init(tmr);
    }
    slot->next = slot->prev = slot;
  }
}

void
MinTmr_Cancel_(MinTmr* tmr)
{
  ZF_LOGD("? Cancel(%p)", tmr);
  tmr->next->prev = tmr->prev;
  tmr->prev->next = tmr->next;
  MinTmr_Init(tmr);
}

bool
MinTmr_After(MinTmr* tmr, TscDuration after, MinSched* sched)
{
  MinTmr_Cancel(tmr);
  MinSched_Trigger(sched);

  uint64_t nSlotsAway = after / sched->interval + 1;
  if (unlikely(nSlotsAway >= sched->nSlots)) {
    ZF_LOGW("%p After(%p, %" PRIu64 ") too-far nSlotsAway=%" PRIu64,
            sched,
            tmr,
            after,
            nSlotsAway);
    return false;
  }

  uint16_t slotNum = (sched->lastSlot + nSlotsAway) & sched->slotMask;
  ZF_LOGD("%p After(%p, %" PRIu64 ") slot=%" PRIu16 " last=%" PRIu16,
          sched,
          tmr,
          after,
          slotNum,
          sched->lastSlot);
  MinTmr* slot = &sched->slot[slotNum];
  tmr->next = slot->next;
  tmr->next->prev = tmr;
  slot->next = tmr;
  tmr->prev = slot;
  return true;
}
