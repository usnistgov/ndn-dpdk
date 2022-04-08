#include "mintmr.h"

#include "../core/logger.h"

N_LOG_INIT(MinTmr);

MinSched*
MinSched_New(int nSlotBits, TscDuration interval, MinTmrCb cb, uintptr_t ctx)
{
  uint32_t nSlots = 1 << nSlotBits;
  NDNDPDK_ASSERT(nSlots != 0);

  MinSched* sched = rte_zmalloc("MinSched", sizeof(MinSched) + nSlots * sizeof(MinTmr), 0);
  sched->interval = interval;
  sched->cb = cb;
  sched->ctx = ctx;
  sched->nSlots = nSlots;
  sched->slotMask = nSlots - 1;
  sched->lastSlot = nSlots - 1;
  sched->nextTime = rte_get_tsc_cycles();

  N_LOGI("New sched=%p slots=%" PRIu16 " interval=%" PRIu64 " cb=%p", sched, sched->nSlots,
         sched->interval, cb);

  for (uint32_t i = 0; i < nSlots; ++i) {
    CDS_INIT_LIST_HEAD(&sched->slot[i]);
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
    N_LOGV("Trigger sched=%p slot=%" PRIu16 " time=%" PRIu64 " now=%" PRIu64, sched,
           sched->lastSlot, sched->nextTime, now);
    sched->nextTime += sched->interval;

    struct cds_list_head* pos;
    struct cds_list_head* p;
    cds_list_for_each_safe (pos, p, &sched->slot[sched->lastSlot]) {
      MinTmr* tmr = cds_list_entry(pos, MinTmr, h);
      cds_list_del_init(pos);
      ++sched->nTriggered;
      sched->cb(tmr, sched->ctx);
    }
  }
}

void
MinTmr_Cancel_(MinTmr* tmr)
{
  N_LOGD("Cancel tmr=%p", tmr);
  cds_list_del_init(&tmr->h);
}

bool
MinTmr_After(MinTmr* tmr, TscDuration after, MinSched* sched)
{
  if (likely(tmr->h.next != NULL)) {
    cds_list_del(&tmr->h);
  }

  uint64_t nSlotsAway = RTE_MAX(after, 0) / sched->interval + 1;
  if (unlikely(nSlotsAway >= sched->nSlots)) {
    N_LOGW("After(too-far) sched=%p tmr=%p after=%" PRId64 " nSlotsAway=%" PRIu64, sched, tmr,
           after, nSlotsAway);
    MinTmr_Init(tmr);
    return false;
  }

  uint32_t slotNum = (sched->lastSlot + nSlotsAway) & sched->slotMask;
  N_LOGD("After sched=%p tmr=%p after=%" PRId64 " slot=%" PRIu16 " last=%" PRIu16, sched, tmr,
         after, slotNum, sched->lastSlot);
  cds_list_add_tail(&tmr->h, &sched->slot[slotNum]);
  return true;
}
