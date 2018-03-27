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
      ZF_LOGD("^ dn-expired=%" PRI_FaceId, dn->face);
      continue;
    }
    Face* dnFace = FaceTable_GetFace(fwd->ft, dn->face);
    if (unlikely(dnFace == NULL)) {
      continue;
    }

    Packet* outNpkt = ClonePacket(npkt, fwd->headerMp, fwd->indirectMp);
    ZF_LOGD("^ data-to=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64,
            dnFace->id, outNpkt, dn->token);
    if (likely(outNpkt != NULL)) {
      Packet_GetLpL3Hdr(outNpkt)->pitToken = dn->token;
      Face_Tx(dnFace, outNpkt);
    }
  }
}

void
FwFwd_RxData(FwFwd* fwd, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;

  ZF_LOGD("data-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64, pkt->port,
          npkt, token);

  PitResult pitFound = Pit_FindByData(fwd->pit, npkt);
  switch (PitResult_GetKind(pitFound)) {
    case PIT_FIND_NONE:
      FwFwd_RxDataUnsolicited(fwd, npkt);
      return;
    case PIT_FIND_PIT0:
      FwFwd_RxDataSatisfy(fwd, npkt, PitFindResult_GetPitEntry0(pitFound));
      break;
    case PIT_FIND_PIT1:
      FwFwd_RxDataSatisfy(fwd, npkt, PitFindResult_GetPitEntry1(pitFound));
      break;
    case PIT_FIND_PIT01:
      // TODO send Data to each downstream only once
      FwFwd_RxDataSatisfy(fwd, npkt, PitFindResult_GetPitEntry0(pitFound));
      FwFwd_RxDataSatisfy(fwd, npkt, PitFindResult_GetPitEntry1(pitFound));
      break;
  }

  // insert to CS
  Cs_Insert(fwd->cs, npkt, pitFound);
}
