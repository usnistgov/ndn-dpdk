#include "input-demux.h"

#include "../core/logger.h"

N_LOG_INIT(InputDemux);

static void
InputDemux_Drop(InputDemux* demux, Packet* npkt, const char* reason)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD("Drop(%s) %s-from=%" PRI_FaceID " npkt=%p token=" PRI_LpPitToken, reason,
         PktType_ToString(Packet_GetType(npkt)), pkt->port, npkt, LpPitToken_Fmt(token));

  ++demux->nDrops;
  rte_pktmbuf_free(pkt);
}

static void
InputDemux_PassTo(InputDemux* demux, Packet* npkt, uint8_t index)
{
  InputDemuxDest* dest = &demux->dest[index];
  if (unlikely(index >= MaxInputDemuxDest || dest->queue == NULL)) {
    InputDemux_Drop(demux, npkt, "no-dest");
    return;
  }

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD("PassTo %s-from=%" PRI_FaceID " npkt=%p token=" PRI_LpPitToken " dest-index=%" PRIu8,
         PktType_ToString(Packet_GetType(npkt)), pkt->port, npkt, LpPitToken_Fmt(token), index);

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
  uint8_t index = Ndtt_Lookup(demux->ndtt, name);
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt, const PName* name)
{
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  if (unlikely(token->length <= demux->byToken.offset)) {
    InputDemux_Drop(demux, npkt, "token-too-short");
    return;
  }

  uint8_t index = token->value[demux->byToken.offset];
  InputDemux_PassTo(demux, npkt, index);
}
