#include "fwd.h"
#include "token.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

static void
FwFwd_RxInterestMissCs(FwFwd* fwd, PitEntry* pitEntry, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceId inFace = pkt->port;
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // insert DN record
  int dnIndex = PitEntry_DnRxInterest(fwd->pit, pitEntry, npkt);
  if (unlikely(dnIndex < 0)) {
    ZF_LOGD("^ pit-entry=%p drop=PitDn-full", pitEntry);
    rte_pktmbuf_free(pkt);
    return;
  }
  ZF_LOGD("^ pit-entry=%p pit-key=%s", pitEntry,
          PitEntry_ToDebugString(pitEntry));
  npkt = NULL; // npkt is owned/freed by pitEntry

  // query FIB, multicast the Interest to every nexthop except inFace
  rcu_read_lock();
  // TODO query with forwarding hint
  const FibEntry* fibEntry = Fib_Lpm(fwd->fib, &interest->name);
  if (unlikely(fibEntry == NULL)) {
    ZF_LOGD("^ drop=no-FIB-match");
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
      ZF_LOGD("^ drop=PitUp-full");
      break;
    }
    if (unlikely(outNpkt == NULL)) {
      ZF_LOGD("^ drop=interest-alloc-error");
      break;
    }

    uint64_t token =
      FwToken_New(fwd->id, Pit_GetEntryToken(fwd->pit, pitEntry));
    Packet_InitLpL3Hdr(outNpkt)->pitToken = token;

    Face* outFace = FaceTable_GetFace(fwd->ft, nh);
    if (unlikely(outFace == NULL)) {
      continue;
    }
    ZF_LOGD("^ interest-to=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64, nh,
            outNpkt, token);
    Face_Tx(outFace, outNpkt);
  }
  rcu_read_unlock();
}

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  PInterest* interest = Packet_GetInterestHdr(npkt);

  ZF_LOGD("interest-from=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          pkt->port, npkt, token);

  PitInsertResult pitIns = Pit_Insert(fwd->pit, interest);
  switch (PitInsertResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1: {
      PitEntry* pitEntry = PitInsertResult_GetPitEntry(pitIns);
      return FwFwd_RxInterestMissCs(fwd, pitEntry, npkt);
    }
    case PIT_INSERT_CS: {
      CsEntry* csEntry = PitInsertResult_GetCsEntry(pitIns);
      ZF_LOGD("^ cs-entry=%p", csEntry);
      break;
    }
    case PIT_INSERT_FULL:
      ZF_LOGD("^ drop=PIT-full");
      rte_pktmbuf_free(pkt);
      break;
    default:
      assert(false); // no other cases
      break;
  }
}
