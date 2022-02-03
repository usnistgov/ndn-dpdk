#include "input-demux.h"

#include "../core/logger.h"

N_LOG_INIT(InputDemux);

__attribute__((nonnull)) static void
InputDemux_Drop(InputDemux* demux, Packet* npkt, const char* reason)
{
  FaceID face = Packet_ToMbuf(npkt)->port;
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD("Drop(%s) %s-from=%" PRI_FaceID " npkt=%p token=" PRI_LpPitToken, reason,
         PktType_ToString(Packet_GetType(npkt)), face, npkt, LpPitToken_Fmt(token));

  ++demux->nDrops;
  Packet_Free(npkt);
}

__attribute__((nonnull)) static __rte_always_inline void
InputDemux_PassTo(InputDemux* demux, Packet* npkt, uint8_t index)
{
  InputDemuxDest* dest = &demux->dest[index];
  if (unlikely(index >= MaxInputDemuxDest || dest->queue == NULL)) {
    InputDemux_Drop(demux, npkt, "no-dest");
    return;
  }

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceID face = pkt->port;
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD("PassTo %s-from=%" PRI_FaceID " npkt=%p token=" PRI_LpPitToken " dest-index=%" PRIu8,
         PktType_ToString(Packet_GetType(npkt)), face, npkt, LpPitToken_Fmt(token), index);

  uint32_t nRej = PktQueue_PushPlain(dest->queue, &pkt, 1);
  if (unlikely(nRej > 0)) {
    ++dest->nDropped;
    Packet_Free(npkt);
  } else {
    ++dest->nQueued;
  }
}

void
InputDemux_DispatchDrop(InputDemux* demux, Packet* npkt, const PName* name)
{
  InputDemux_Drop(demux, npkt, "op-drop");
}

__attribute__((nonnull)) static void
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt, const PName* name)
{
  InputDemux_PassTo(demux, npkt, 0);
}

__attribute__((nonnull)) static void
InputDemux_DispatchRoundrobinDiv(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint8_t index = (++demux->div.i) % demux->div.n;
  InputDemux_PassTo(demux, npkt, index);
}

__attribute__((nonnull)) static void
InputDemux_DispatchRoundrobinMask(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint8_t index = (++demux->div.i) & demux->div.mask;
  InputDemux_PassTo(demux, npkt, index);
}

__attribute__((nonnull)) static __rte_always_inline uint64_t
InputDemux_ComputeGenericHash(const PName* name)
{
  return PName_ComputePrefixHash(name, RTE_MIN(name->nComps, (uint16_t)name->firstNonGeneric));
}

__attribute__((nonnull)) static void
InputDemux_DispatchGenericHashDiv(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint64_t hash = InputDemux_ComputeGenericHash(name);
  uint8_t index = hash % demux->div.n;
  InputDemux_PassTo(demux, npkt, index);
}

__attribute__((nonnull)) static void
InputDemux_DispatchGenericHashMask(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint64_t hash = InputDemux_ComputeGenericHash(name);
  uint8_t index = hash & demux->div.mask;
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_SetDispatchDiv(InputDemux* demux, uint32_t nDest, bool byGenericHash)
{
  if (nDest <= 1) {
    demux->dispatch = InputDemux_DispatchToFirst;
  } else if (RTE_IS_POWER_OF_2(nDest)) {
    demux->div.mask = nDest - 1;
    demux->dispatch =
      byGenericHash ? InputDemux_DispatchGenericHashMask : InputDemux_DispatchRoundrobinMask;
  } else {
    demux->div.n = nDest;
    demux->dispatch =
      byGenericHash ? InputDemux_DispatchGenericHashDiv : InputDemux_DispatchRoundrobinDiv;
  }
}

__attribute__((nonnull)) static void
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt, const PName* name)
{
  uint8_t index = NdtQuerier_Lookup(demux->ndq, name);
  InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_SetDispatchByNdt(InputDemux* demux, NdtQuerier* ndq)
{
  demux->ndq = ndq;
  demux->dispatch = InputDemux_DispatchByNdt;
}

__attribute__((nonnull)) static void
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

void
InputDemux_SetDispatchByToken(InputDemux* demux, uint8_t offset)
{
  demux->byToken.offset = offset;
  demux->dispatch = InputDemux_DispatchByToken;
}
