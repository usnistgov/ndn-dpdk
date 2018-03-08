#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwInterest);

static void
FwFwd_RxInterestMissCs(FwFwd* fwd, PitEntry* pitEntry, Packet* npkt)
{
  FaceId inFace = Packet_ToMbuf(npkt)->port;
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // insert DN record
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

  // query FIB, multicast the Interest to every nexthop except inFace
  rcu_read_lock();
  // TODO consider forwarding hint
  const FibEntry* fibEntry = Fib_Lpm(fwd->fib, &interest->name);
  if (unlikely(fibEntry == NULL)) {
    ZF_LOGV("%" PRIu8 " %s FIB-no-match", fwd->id,
            PitEntry_ToDebugString(pitEntry));
    rcu_read_unlock();
    return;
  }

  for (uint8_t i = 0; i < fibEntry->nNexthops; ++i) {
    FaceId nh = fibEntry->nexthops[i];
    if (unlikely(nh == inFace)) {
      continue;
    }

    // TODO create other strategies
    Packet* outNpkt;
    int upIndex = PitEntry_UpTxInterest(fwd->pit, pitEntry, nh, &outNpkt);
    if (unlikely(upIndex < 0)) {
      break;
    }
    if (unlikely(outNpkt == NULL)) {
      break;
    }
    Face* outFace = FaceTable_GetFace(fwd->ft, nh);
    if (unlikely(outFace == NULL)) {
      continue;
    }
    ZF_LOGV("%" PRIu8 " %s nh=%" PRI_FaceId " upIndex=%d", fwd->id,
            PitEntry_ToDebugString(pitEntry), nh, upIndex);
    Face_Tx(outFace, outNpkt);
  }
  rcu_read_unlock();
}

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("%" PRIu8 " %p RxInterest", fwd->id, npkt);

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
