#include "fwd.h"
#include "token.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwInterest);

static void
FwFwd_RxInterestMissCs(FwFwd* fwd, PitEntry* pitEntry, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceId inFace = pkt->port;
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // insert DN record
  int dnIndex = PitEntry_DnRxInterest(fwd->pit, pitEntry, npkt);
  if (dnIndex < 0) {
    ZF_LOGW("%" PRIu8 " %s PitDn-full", fwd->id,
            PitEntry_ToDebugString(pitEntry));
    rte_pktmbuf_free(pkt);
    return;
  }
  npkt = NULL; // npkt is owned/freed by pitEntry
  ZF_LOGV("%" PRIu8 " %s dnIndex=%d", fwd->id, PitEntry_ToDebugString(pitEntry),
          dnIndex);

  // query FIB, multicast the Interest to every nexthop except inFace
  rcu_read_lock();
  // TODO query with forwarding hint
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

    FwToken token = { 0 };
    token.fwdId = fwd->id;
    token.pccToken = Pit_AddToken(fwd->pit, pitEntry);
    Packet_InitLpL3Hdr(outNpkt)->pitToken = token.token;

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
  PInterest* interest = Packet_GetInterestHdr(npkt);

  PitInsertResult pitIns = Pit_Insert(fwd->pit, interest);
  switch (PitInsertResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1:
      return FwFwd_RxInterestMissCs(fwd, PitInsertResult_GetPitEntry(pitIns),
                                    npkt);
    case PIT_INSERT_CS:
      assert(false); // not implemented
      break;
    case PIT_INSERT_FULL:
      rte_pktmbuf_free(Packet_ToMbuf(npkt));
      break;
    default:
      assert(false); // no other cases
      break;
  }
}
