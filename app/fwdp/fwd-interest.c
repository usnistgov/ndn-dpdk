#include "fwd.h"
#include "token.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

static void
FwFwd_RxInterestMissCs(FwFwd* fwd, PitEntry* pitEntry, Packet* npkt,
                       const FibEntry* fibEntry)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
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

  for (uint8_t i = 0; i < fibEntry->nNexthops; ++i) {
    FaceId nh = fibEntry->nexthops[i];
    if (unlikely(nh == pkt->port)) {
      continue;
    }

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
}

static void
FwFwd_RxInterestHitCs(FwFwd* fwd, CsEntry* csEntry, Packet* npkt, Face* dnFace)
{
  uint64_t dnToken = Packet_GetLpL3Hdr(npkt)->pitToken;
  Packet* outNpkt = ClonePacket(csEntry->data, fwd->headerMp, fwd->indirectMp);
  ZF_LOGD("^ cs-entry=%p data-to=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          csEntry, dnFace->id, outNpkt, dnToken);
  if (likely(outNpkt != NULL)) {
    Packet_GetLpL3Hdr(outNpkt)->pitToken = dnToken;
    Face_Tx(dnFace, outNpkt);
  }
}

void
FwFwd_RxInterest(FwFwd* fwd, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  PInterest* interest = Packet_GetInterestHdr(npkt);
  Face* dnFace = FaceTable_GetFace(fwd->ft, pkt->port);
  assert(dnFace != NULL); // XXX could fail if face fails during forwarding

  ZF_LOGD("interest-from=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
          dnFace->id, npkt, token);

  rcu_read_lock();

  // query FIB; TODO query with forwarding hint
  const FibEntry* fibEntry = Fib_Lpm(fwd->fib, &interest->name);
  if (unlikely(fibEntry == NULL)) {
    // Nack if no FIB match
    ZF_LOGD("^ drop=no-FIB-match nack-to=%" PRI_FaceId, dnFace->id);
    MakeNack(npkt, NackReason_NoRoute);
    Face_Tx(dnFace, npkt);
    rcu_read_unlock();
    return;
  }

  // TODO insert PIT entry with forwarding hint
  PitResult pitIns = Pit_Insert(fwd->pit, npkt);
  switch (PitResult_GetKind(pitIns)) {
    case PIT_INSERT_PIT0:
    case PIT_INSERT_PIT1: {
      PitEntry* pitEntry = PitInsertResult_GetPitEntry(pitIns);
      FwFwd_RxInterestMissCs(fwd, pitEntry, npkt, fibEntry);
      break;
    }
    case PIT_INSERT_CS: {
      CsEntry* csEntry = PitInsertResult_GetCsEntry(pitIns);
      FwFwd_RxInterestHitCs(fwd, csEntry, npkt, dnFace);
      break;
    }
    case PIT_INSERT_FULL:
      ZF_LOGD("^ drop=PIT-full nack-to=%" PRI_FaceId, dnFace->id);
      MakeNack(npkt, NackReason_Congestion);
      Face_Tx(dnFace, npkt);
      break;
    default:
      assert(false); // no other cases
      break;
  }

  rcu_read_unlock();
}
