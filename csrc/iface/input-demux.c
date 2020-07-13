#include "input-demux.h"

#include "../core/logger.h"

INIT_ZF_LOG(InputDemux);

static void
InputDemux_Drop(InputDemux* demux, Packet* npkt, const char* reason)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGD("%s-from=%" PRI_FaceID " npkt=%p token=%016" PRIx64 " drop=%s",
          Pkt_ToString(Packet_GetType(npkt)), pkt->port, npkt, Packet_GetLpL3Hdr(npkt)->pitToken,
          reason);

  ++demux->nDrops;
  rte_pktmbuf_free(pkt);
}

static void
InputDemux_PassTo(InputDemux* demux, Packet* npkt, uint8_t index)
{
  InputDemuxDest* dest = &demux->dest[index];
  if (unlikely(index >= INPUTDEMUX_DEST_MAX || dest->queue == NULL)) {
    InputDemux_Drop(demux, npkt, "no-dest");
    return;
  }

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  ZF_LOGD("%s-from=%" PRI_FaceID " npkt=%p token=%016" PRIx64 " dest-index=%" PRIu8,
          Pkt_ToString(Packet_GetType(npkt)), pkt->port, npkt, Packet_GetLpL3Hdr(npkt)->pitToken,
          index);

  uint32_t nRej = PktQueue_PushPlain(dest->queue, &pkt, 1);
  dest->nDropped += nRej;
  dest->nQueued += 1 - nRej;
}

void
InputDemux_DispatchDrop(InputDemux* demux, Packet* npkt, const PName* name)
{
  InputDemux_Drop(demux, npkt, "op-drop");
}

void
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt, const PName* name)
{
  InputDemux_PassTo(demux, npkt, 0);
}

void
InputDemux_DispatchRoundrobinDiv(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint8_t index = (++demux->roundrobin.i) % demux->roundrobin.n;
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_DispatchRoundrobinMask(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint8_t index = (++demux->roundrobin.i) & demux->roundrobin.n;
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint8_t index = Ndtt_Lookup(demux->ndt, demux->ndtt, name);
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  if (unlikely(token == 0)) {
    InputDemux_Drop(demux, npkt, "no-token");
    return;
  }

  uint8_t index = token >> 56;
  InputDemux_PassTo(demux, npkt, index);
}
