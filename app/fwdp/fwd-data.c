#include "fwd.h"

#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

static void
FwFwd_RxDataUnsolicited(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("^ drop=unsolicited");
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

static void
FwFwd_RxDataSatisfy(FwFwd* fwd, Packet* npkt, PitEntry* pitEntry)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGD("^ pit-entry=%p pit-key=%s", pitEntry,
          PitEntry_ToDebugString(pitEntry));

  for (int index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
    PitDn* dn = &pitEntry->dns[index];
    if (unlikely(dn->face == FACEID_INVALID)) {
      if (index == 0) {
        ZF_LOGD("^ drop=PitDn-empty");
      }
      break;
    }
    if (unlikely(dn->expiry < pkt->timestamp)) {
      ZF_LOGV("^ dn-expired=%" PRI_FaceId, dn->face);
      continue;
    }
    Face* outFace = FaceTable_GetFace(fwd->ft, dn->face);
    if (unlikely(outFace == NULL)) {
      continue;
    }

    Packet* outNpkt = ClonePacket(npkt, fwd->headerMp, fwd->indirectMp);
    ZF_LOGD("^ data-to=%" PRI_FaceId " npkt=%p", dn->face, outNpkt);
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
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  ZF_LOGD("data-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64, pkt->port,
          npkt, token);

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
