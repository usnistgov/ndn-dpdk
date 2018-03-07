#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwInterest);

static void
FwFwd_RxInterestMissCs(FwFwd* fwd, PitEntry* pitEntry, Packet* npkt)
{
  int dnIndex = PitEntry_DnRxInterest(fwd->pit, pitEntry, npkt);
  if (dnIndex < 0) {
    ZF_LOGW("%" PRIu8 " %s PitDn-full", fwd->id,
            PitEntry_ToDebugString(pitEntry));
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
    return;
  }
  npkt = NULL; // npkt is owned/freed by pitEntry
  ZF_LOGV("%" PRIu8 " %s dnIndex=%d", fwd->id, PitEntry_ToDebugString(pitEntry),
          dnIndex);

  // TODO
}

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("%" PRIu8 " RxInterest", fwd->id);

  PitInsertResult pitIns = Pit_Insert(fwd->pit, npkt);
  switch (PitInsertResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1:
      return FwFwd_RxInterestMissCs(fwd, PitInsertResult_GetPitEntry(pitIns),
                                    npkt);
    default:
      assert(false); // will not happen before implementing RxData procedure
  }
}
