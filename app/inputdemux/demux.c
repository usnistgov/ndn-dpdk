#include "demux.h"

#include "../../core/logger.h"

INIT_ZF_LOG(InputDemux);

static void
InputDemux_Drop(InputDemux* demux, Packet* npkt)
{
  ++demux->nNoDest;
  rte_pktmbuf_free(Packet_ToMbuf(npkt));
}

static void
InputDemux_PassTo(InputDemux* demux, Packet* npkt, uint8_t index)
{
  InputDemuxDest* dest = &demux->dest[index];
  if (unlikely(index >= INPUTDEMUX_DEST_MAX || dest->queue == NULL)) {
    InputDemux_Drop(demux, npkt);
    return;
  }

  struct rte_mbuf* pkts[1] = { Packet_ToMbuf(npkt) };
  uint32_t nRej = PktQueue_PushPlain(dest->queue, pkts, 1);
  dest->nDropped += nRej;
  dest->nQueued += 1 - nRej;
}

void
InputDemux_DispatchDrop(InputDemux* demux, Packet* npkt, const Name* name)
{
  InputDemux_Drop(demux, npkt);
}

void
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt, const Name* name)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p token=%016" PRIx64 " dest-index=0",
          L3PktType_ToString(Packet_GetL3PktType(npkt)),
          pkt->port,
          npkt,
          Packet_GetLpL3Hdr(npkt)->pitToken);
  InputDemux_PassTo(demux, npkt, 0);
}

void
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt, const Name* name)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint8_t index = Ndtt_Lookup(demux->ndt, demux->ndtt, name);
  ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p token=%016" PRIx64
          " dest-index=%" PRIu8,
          L3PktType_ToString(Packet_GetL3PktType(npkt)),
          pkt->port,
          npkt,
          Packet_GetLpL3Hdr(npkt)->pitToken,
          index);
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt, const Name* name)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  if (unlikely(token == 0)) {
    InputDemux_Drop(demux, npkt);
    return;
  }

  uint8_t index = token >> 56;
  ZF_LOGD("%s-from=%" PRI_FaceId " npkt=%p token=%016" PRIx64
          " dest-index=%" PRIu8,
          L3PktType_ToString(Packet_GetL3PktType(npkt)),
          pkt->port,
          npkt,
          token,
          index);
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux3_FaceRx(FaceRxBurst* burst, void* demux0)
{
  InputDemux3* demux3 = (InputDemux3*)demux0;
  for (uint16_t i = 0; i < burst->nInterests; ++i) {
    Packet* npkt = FaceRxBurst_GetInterest(burst, i);
    PInterest* interest = Packet_GetInterestHdr(npkt);
    InputDemux_Dispatch(&demux3->interest, npkt, &interest->name);
  }
  for (uint16_t i = 0; i < burst->nData; ++i) {
    Packet* npkt = FaceRxBurst_GetData(burst, i);
    PData* data = Packet_GetDataHdr(npkt);
    InputDemux_Dispatch(&demux3->data, npkt, &data->name);
  }
  for (uint16_t i = 0; i < burst->nNacks; ++i) {
    Packet* npkt = FaceRxBurst_GetNack(burst, i);
    PNack* nack = Packet_GetNackHdr(npkt);
    InputDemux_Dispatch(&demux3->nack, npkt, &nack->interest.name);
  }
}
