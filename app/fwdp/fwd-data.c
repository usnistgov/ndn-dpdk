#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwData);

static void
FwFwd_RxDataUnsolicited(FwFwd* fwd, Packet* npkt)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  ZF_LOGD("%" PRIu8 " %p RxDataUnsolicited token=%" PRIx64, fwd->id, npkt,
          token);
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

static void
FwFwd_RxDataSatisfy(FwFwd* fwd, Packet* npkt, PitEntry* pitEntry)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);

  for (int index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
    PitDn* dn = &pitEntry->dns[index];
    if (dn->face == FACEID_INVALID) {
      break;
    }
    if (dn->expiry < pkt->timestamp) {
      ZF_LOGV("%" PRIu8 " %s dn[%i]=%" PRI_FaceId " expired", fwd->id,
              PitEntry_ToDebugString(pitEntry), index, dn->face);
      continue;
    }
    Face* outFace = FaceTable_GetFace(fwd->ft, dn->face);
    if (unlikely(outFace == NULL)) {
      continue;
    }

    Packet* outNpkt = ClonePacket(npkt, fwd->headerMp, fwd->indirectMp);
    ZF_LOGV("%" PRIu8 " %s dn[%i]=%" PRI_FaceId " TxData=%p", fwd->id,
            PitEntry_ToDebugString(pitEntry), index, dn->face, outNpkt);
    if (likely(outNpkt != NULL)) {
      Face_Tx(outFace, outNpkt);
    }
  }

  Pit_Erase(fwd->pit, pitEntry);
}

void
FwFwd_RxData(FwFwd* fwd, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGD("%" PRIu8 " %p RxData from=%" PRI_FaceId, fwd->id, npkt, pkt->port);

  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  PitFindResult pitFound;
  Pit_Find(fwd->pit, token, &pitFound);

  if (unlikely(pitFound.nMatches == 0)) {
    FwFwd_RxDataUnsolicited(fwd, npkt);
    return;
  }

  for (uint8_t i = 0; i < pitFound.nMatches; ++i) {
    // TODO if both PIT entries have same downstream face,
    //      Data should be sent only once
    FwFwd_RxDataSatisfy(fwd, npkt, pitFound.matches[i]);
  }

  // TODO insert to CS
  rte_pktmbuf_free(pkt);
}
