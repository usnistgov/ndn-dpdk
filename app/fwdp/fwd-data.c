#include "fwd.h"

#include "../../container/pcct/pit-dn-up-it.h"
#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

static void
FwFwd_DataUnsolicited(FwFwd* fwd, Packet* npkt)
{
  ZF_LOGD("^ drop=unsolicited");
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

static void
FwFwd_DataSatisfy(FwFwd* fwd, Packet* npkt, PitEntry* pitEntry)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGD("^ pit-entry=%p pit-key=%s", pitEntry,
          PitEntry_ToDebugString(pitEntry));

  PitDnIt it;
  for (PitDnIt_Init(&it, pitEntry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
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
    if (unlikely(Face_IsDown(dn->face))) {
      ZF_LOGD("^ no-data-to=%" PRI_FaceId " drop=face-down", dn->face);
      continue;
    }

    Packet* outNpkt = ClonePacket(npkt, fwd->headerMp, fwd->indirectMp);
    ZF_LOGD("^ data-to=%" PRI_FaceId " npkt=%p dn-token=%016" PRIx64, dn->face,
            outNpkt, dn->token);
    if (likely(outNpkt != NULL)) {
      Packet_GetLpL3Hdr(outNpkt)->pitToken = dn->token;
      Face_Tx(dn->face, outNpkt);
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
      FwFwd_DataUnsolicited(fwd, npkt);
      return;
    case PIT_FIND_PIT0:
      FwFwd_DataSatisfy(fwd, npkt, PitFindResult_GetPitEntry0(pitFound));
      break;
    case PIT_FIND_PIT1:
      FwFwd_DataSatisfy(fwd, npkt, PitFindResult_GetPitEntry1(pitFound));
      break;
    case PIT_FIND_PIT01:
      // XXX if both PIT entries have the same downstream, Data is sent twice
      FwFwd_DataSatisfy(fwd, npkt, PitFindResult_GetPitEntry0(pitFound));
      FwFwd_DataSatisfy(fwd, npkt, PitFindResult_GetPitEntry1(pitFound));
      break;
  }

  // insert to CS
  Cs_Insert(fwd->cs, npkt, pitFound);
}
