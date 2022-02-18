#include "input-demux.h"

#include "../core/logger.h"

N_LOG_INIT(InputDemux);

__attribute__((nonnull)) static inline bool
InputDemux_Drop(InputDemux* demux, Packet* npkt, const char* reason)
{
  FaceID face = Packet_ToMbuf(npkt)->port;
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD("Drop(%s) %s-from=%" PRI_FaceID " npkt=%p token=" PRI_LpPitToken, reason,
         PktType_ToString(Packet_GetType(npkt)), face, npkt, LpPitToken_Fmt(token));

  ++demux->nDrops;
  return false;
}

__attribute__((nonnull)) static __rte_always_inline bool
InputDemux_PassTo(InputDemux* demux, Packet* npkt, uint8_t index)
{
  InputDemuxDest* dest = &demux->dest[index];
  if (unlikely(index >= MaxInputDemuxDest || dest->queue == NULL)) {
    return InputDemux_Drop(demux, npkt, "no-dest");
  }

  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceID face = pkt->port;
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  N_LOGD("PassTo %s-from=%" PRI_FaceID " npkt=%p token=" PRI_LpPitToken " dest-index=%" PRIu8,
         PktType_ToString(Packet_GetType(npkt)), face, npkt, LpPitToken_Fmt(token), index);

  uint32_t nRej = PktQueue_PushPlain(dest->queue, &pkt, 1);
  if (unlikely(nRej > 0)) {
    ++dest->nDropped;
    return false;
  }
  ++dest->nQueued;
  return true;
}

__attribute__((nonnull)) static bool
InputDemux_DispatchDrop(InputDemux* demux, Packet* npkt)
{
  return InputDemux_Drop(demux, npkt, "op-drop");
}

__attribute__((nonnull)) static bool
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt)
{
  return InputDemux_PassTo(demux, npkt, 0);
}

__attribute__((nonnull)) static bool
InputDemux_DispatchRoundrobinDiv(InputDemux* demux, Packet* npkt)
{
  uint8_t index = (++demux->div.i) % demux->div.n;
  return InputDemux_PassTo(demux, npkt, index);
}

__attribute__((nonnull)) static bool
InputDemux_DispatchRoundrobinMask(InputDemux* demux, Packet* npkt)
{
  uint8_t index = (++demux->div.i) & demux->div.mask;
  return InputDemux_PassTo(demux, npkt, index);
}

__attribute__((nonnull)) static __rte_always_inline uint64_t
InputDemux_ComputeGenericHash(Packet* npkt)
{
  const PName* name = Packet_GetName(npkt);
  return PName_ComputePrefixHash(name, RTE_MIN(name->nComps, (uint16_t)name->firstNonGeneric));
}

__attribute__((nonnull)) static bool
InputDemux_DispatchGenericHashDiv(InputDemux* demux, Packet* npkt)
{
  uint64_t hash = InputDemux_ComputeGenericHash(npkt);
  uint8_t index = hash % demux->div.n;
  return InputDemux_PassTo(demux, npkt, index);
}

__attribute__((nonnull)) static bool
InputDemux_DispatchGenericHashMask(InputDemux* demux, Packet* npkt)
{
  uint64_t hash = InputDemux_ComputeGenericHash(npkt);
  uint8_t index = hash & demux->div.mask;
  return InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_SetDispatchDiv(InputDemux* demux, uint32_t nDest, bool byGenericHash)
{
  if (nDest <= 1) {
    demux->dispatch = InputDemuxFuncToFirst;
  } else if (rte_is_power_of_2(nDest)) {
    demux->div.mask = nDest - 1;
    demux->dispatch = byGenericHash ? InputDemuxFuncGenericHashMask : InputDemuxFuncRoundrobinMask;
  } else {
    demux->div.n = nDest;
    demux->dispatch = byGenericHash ? InputDemuxFuncGenericHashDiv : InputDemuxFuncRoundrobinDiv;
  }
}

__attribute__((nonnull)) static bool
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt)
{
  uint8_t index = NdtQuerier_Lookup(demux->ndq, Packet_GetName(npkt));
  return InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_SetDispatchByNdt(InputDemux* demux, NdtQuerier* ndq)
{
  demux->ndq = ndq;
  demux->dispatch = InputDemuxFuncByNdt;
}

__attribute__((nonnull)) static bool
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt)
{
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  if (unlikely(token->length <= demux->byToken.offset)) {
    return InputDemux_Drop(demux, npkt, "token-too-short");
  }

  uint8_t index = token->value[demux->byToken.offset];
  return InputDemux_PassTo(demux, npkt, index);
}

void
InputDemux_SetDispatchByToken(InputDemux* demux, uint8_t offset)
{
  demux->byToken.offset = offset;
  demux->dispatch = InputDemuxFuncByToken;
}

const InputDemux_DispatchFunc InputDemux_DispatchFuncTable[] = {
  [InputDemuxFuncDrop] = InputDemux_DispatchDrop,
  [InputDemuxFuncToFirst] = InputDemux_DispatchToFirst,
  [InputDemuxFuncRoundrobinDiv] = InputDemux_DispatchRoundrobinDiv,
  [InputDemuxFuncRoundrobinMask] = InputDemux_DispatchRoundrobinMask,
  [InputDemuxFuncGenericHashDiv] = InputDemux_DispatchGenericHashDiv,
  [InputDemuxFuncGenericHashMask] = InputDemux_DispatchGenericHashMask,
  [InputDemuxFuncByNdt] = InputDemux_DispatchByNdt,
  [InputDemuxFuncByToken] = InputDemux_DispatchByToken,
};
