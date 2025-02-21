#include "input-demux.h"

#include "../core/logger.h"

N_LOG_INIT(InputDemux);

/** @brief Drop npkts[lbound:ubound]. */
__attribute__((nonnull)) static __rte_always_inline uint64_t
Drop(InputDemux* demux, Packet** npkts, uint16_t lbound, uint16_t ubound, const char* reason) {
  if (lbound == ubound) {
    return 0;
  }

  if (N_LOG_ENABLED(DEBUG)) {
    for (uint16_t i = lbound; i < ubound; ++i) {
      Packet* npkt = npkts[i];
      FaceID face = Packet_ToMbuf(npkt)->port;
      const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
      N_LOGD("Drop(%s) %s-from=%" PRI_FaceID " npkt=%p token=%s", reason,
             PktType_ToString(Packet_GetType(npkt)), face, npkt, LpPitToken_ToString(token));
    }
  }

  demux->nDrops += ubound - lbound;
  return RTE_GENMASK64(ubound - 1, lbound);
}

/** @brief Pass npkts[lbound:ubound] to destination identified by @p index . */
__attribute__((nonnull)) static __rte_always_inline uint64_t
PassTo(InputDemux* demux, uint8_t index, Packet** npkts, uint16_t lbound, uint16_t ubound) {
  if (lbound == ubound) {
    return 0;
  }

  InputDemuxDest* dest = &demux->dest[index];
  if (unlikely(index >= RTE_DIM(demux->dest) || dest->queue == NULL)) {
    return Drop(demux, npkts, lbound, ubound, "no-dest");
  }

  if (N_LOG_ENABLED(DEBUG)) {
    for (uint16_t i = lbound; i < ubound; ++i) {
      Packet* npkt = npkts[i];
      FaceID face = Packet_ToMbuf(npkt)->port;
      const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
      N_LOGD("PassTo %s-from=%" PRI_FaceID " npkt=%p token=%s dest-index=%" PRIu8,
             PktType_ToString(Packet_GetType(npkt)), face, npkt, LpPitToken_ToString(token), index);
    }
  }

  uint16_t count = ubound - lbound;
  uint32_t nRej = PktQueue_Push(dest->queue, (struct rte_mbuf**)&npkts[lbound], count);
  uint32_t nAccept = count - nRej;
  dest->nQueued += nAccept;
  return Drop(demux, npkts, lbound + nAccept, ubound, "dest-full");
}

/**
 * @brief Pass to destination or drop with reason.
 *
 * If @p dropReason is not NULL and @p index is UINT8_MAX, drop the packets with specified reason.
 * Otherwise, pass the packets to destination identified by @p index .
 */
__attribute__((nonnull(1, 3))) static __rte_always_inline uint64_t
PassToOrDrop(InputDemux* demux, uint8_t index, Packet** npkts, uint16_t lbound, uint16_t ubound,
             const char* dropReason) {
  if (dropReason != NULL && unlikely(index == UINT8_MAX)) {
    return Drop(demux, npkts, lbound, ubound, dropReason);
  }
  return PassTo(demux, index, npkts, lbound, ubound);
}

/**
 * @brief Dispatch packets using a loop.
 * @param getIndex callback function to determine destination index.
 *                 Return UINT8_MAX to drop the packet
 * @param dropReason packet drop reason when @p getIndex drops the packet.
 *                   If NULL, @p getIndex must not drop the packet.
 */
__attribute__((nonnull(1, 2, 4))) static __rte_always_inline uint64_t
DispatchLoop(InputDemux* demux, Packet** npkts, uint16_t count,
             __attribute__((nonnull)) uint8_t (*getIndex)(InputDemux* demux, Packet* npkt),
             const char* dropReason) {
  uint64_t mask = 0;
  uint16_t lbound = 0;
  uint8_t lastIndex = 0;
  for (uint16_t i = 0; i < count; ++i) {
    uint8_t thisIndex = getIndex(demux, npkts[i]);
    if (unlikely(thisIndex != lastIndex)) {
      mask |= PassToOrDrop(demux, lastIndex, npkts, lbound, i, dropReason);
      lbound = i;
      lastIndex = thisIndex;
    }
  }
  mask |= PassToOrDrop(demux, lastIndex, npkts, lbound, count, dropReason);
  return mask;
}

__attribute__((nonnull)) static uint64_t
DispatchDrop(InputDemux* demux, Packet** npkts, uint16_t count) {
  return Drop(demux, npkts, 0, count, "op-drop");
}

__attribute__((nonnull)) static uint64_t
DispatchRoundrobinDiv(InputDemux* demux, Packet** npkts, uint16_t count) {
  uint8_t index = (++demux->div.i) % demux->div.n;
  return PassTo(demux, index, npkts, 0, count);
}

__attribute__((nonnull)) static uint64_t
DispatchRoundrobinMask(InputDemux* demux, Packet** npkts, uint16_t count) {
  uint8_t index = (++demux->div.i) & demux->div.mask;
  return PassTo(demux, index, npkts, 0, count);
}

__attribute__((nonnull)) static __rte_always_inline uint64_t
ComputeGenericHash(Packet* npkt) {
  const PName* name = Packet_GetName(npkt);
  return PName_ComputePrefixHash(name, RTE_MIN(name->nComps, (uint16_t)name->firstNonGeneric));
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
GetIndexFromGenericHashDiv(InputDemux* demux, Packet* npkt) {
  return ComputeGenericHash(npkt) % demux->div.n;
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
GetIndexFromGenericHashMask(InputDemux* demux, Packet* npkt) {
  return ComputeGenericHash(npkt) & demux->div.mask;
}

__attribute__((nonnull)) static uint64_t
DispatchGenericHashDiv(InputDemux* demux, Packet** npkts, uint16_t count) {
  return DispatchLoop(demux, npkts, count, GetIndexFromGenericHashDiv, "");
}

__attribute__((nonnull)) static uint64_t
DispatchGenericHashMask(InputDemux* demux, Packet** npkts, uint16_t count) {
  return DispatchLoop(demux, npkts, count, GetIndexFromGenericHashMask, "");
}

void
InputDemux_SetDispatchDiv(InputDemux* demux, uint32_t nDest, bool byGenericHash) {
  if (nDest <= 1) {
    demux->div.mask = 0;
    demux->dispatch = InputDemuxActRoundrobinMask;
  } else if (rte_is_power_of_2(nDest)) {
    demux->div.mask = nDest - 1;
    demux->dispatch = byGenericHash ? InputDemuxActGenericHashMask : InputDemuxActRoundrobinMask;
  } else {
    demux->div.n = nDest;
    demux->dispatch = byGenericHash ? InputDemuxActGenericHashDiv : InputDemuxActRoundrobinDiv;
  }
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
GetIndexByNdt(InputDemux* demux, Packet* npkt) {
  return NdtQuerier_Lookup(&demux->ndq, Packet_GetName(npkt));
}

__attribute__((nonnull)) static uint64_t
DispatchByNdt(InputDemux* demux, Packet** npkts, uint16_t count) {
  return DispatchLoop(demux, npkts, count, GetIndexByNdt, "");
}

NdtQuerier*
InputDemux_SetDispatchByNdt(InputDemux* demux) {
  demux->dispatch = InputDemuxActByNdt;
  return &demux->ndq;
}

__attribute__((nonnull)) static __rte_always_inline uint8_t
GetIndexByToken(InputDemux* demux, Packet* npkt) {
  const LpPitToken* token = &Packet_GetLpL3Hdr(npkt)->pitToken;
  if (unlikely(token->length <= demux->byToken.offset)) {
    return UINT8_MAX;
  }

  static_assert(MaxInputDemuxDest <= UINT8_MAX, "");
  return token->value[demux->byToken.offset];
}

__attribute__((nonnull)) static uint64_t
DispatchByToken(InputDemux* demux, Packet** npkts, uint16_t count) {
  return DispatchLoop(demux, npkts, count, GetIndexByToken, "token-too-short");
}

void
InputDemux_SetDispatchByToken(InputDemux* demux, uint8_t offset) {
  demux->byToken.offset = offset;
  demux->dispatch = InputDemuxActByToken;
}

const InputDemux_DispatchFunc InputDemux_DispatchJmp[] = {
  [InputDemuxActDrop] = DispatchDrop,
  [InputDemuxActRoundrobinDiv] = DispatchRoundrobinDiv,
  [InputDemuxActRoundrobinMask] = DispatchRoundrobinMask,
  [InputDemuxActGenericHashDiv] = DispatchGenericHashDiv,
  [InputDemuxActGenericHashMask] = DispatchGenericHashMask,
  [InputDemuxActByNdt] = DispatchByNdt,
  [InputDemuxActByToken] = DispatchByToken,
};
