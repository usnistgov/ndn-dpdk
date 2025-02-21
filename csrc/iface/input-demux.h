#ifndef NDNDPDK_IFACE_INPUT_DEMUX_H
#define NDNDPDK_IFACE_INPUT_DEMUX_H

/** @file */

#include "../ndt/ndt.h"
#include "faceid.h"
#include "pktqueue.h"

/** @brief Input demultiplexer dispatch method. */
typedef enum InputDemuxAct {
  InputDemuxActDrop,
  InputDemuxActRoundrobinDiv,
  InputDemuxActRoundrobinMask,
  InputDemuxActGenericHashDiv,
  InputDemuxActGenericHashMask,
  InputDemuxActByNdt,
  InputDemuxActByToken,
} InputDemuxAct;

/** @brief Destination of input packet demultiplexer. */
typedef struct InputDemuxDest {
  PktQueue* queue;
  uint64_t nQueued;
  uint64_t nDropped;
} InputDemuxDest;

/** @brief Input packet demultiplexer for a single packet type. */
typedef struct InputDemux {
  uint64_t nDrops;
  InputDemuxAct dispatch;
  union {
    struct {
      uint32_t i;
      union {
        uint32_t n;
        uint32_t mask;
      };
    } div;
    NdtQuerier ndq;
    struct {
      uint8_t offset;
    } byToken;
  };
  InputDemuxDest dest[MaxInputDemuxDest];
} InputDemux;

typedef uint64_t (*InputDemux_DispatchFunc)(InputDemux* demux, Packet** npkts, uint16_t count);
extern const InputDemux_DispatchFunc InputDemux_DispatchJmp[];

__attribute__((nonnull, returns_nonnull)) NdtQuerier*
InputDemux_SetDispatchByNdt(InputDemux* demux);

__attribute__((nonnull)) void
InputDemux_SetDispatchDiv(InputDemux* demux, uint32_t nDest, bool byGenericHash);

__attribute__((nonnull)) void
InputDemux_SetDispatchByToken(InputDemux* demux, uint8_t offset);

/**
 * @brief Dispatch a burst of L3 packets.
 * @param npkts L3 packets. They shall come from the same face.
 * @return bitset of rejected packets. They must be freed by the caller.
 */
__attribute__((nonnull, warn_unused_result)) static inline uint64_t
InputDemux_Dispatch(InputDemux* demux, Packet** npkts, uint16_t count) {
  NDNDPDK_ASSERT(count > 0 && count <= 64);
  return InputDemux_DispatchJmp[demux->dispatch](demux, npkts, count);
}

/**
 * @brief Append rejected packets to be freed.
 * @param frees vector of mbufs to be freed by the caller.
 * @param[inout] nFree index into @p frees vector.
 * @param npkts vector of L3 packets passed to @c InputDemux_Dispatch .
 * @param mask return value of @c InputDemux_Dispatch .
 */
__attribute__((nonnull)) static inline void
InputDemux_FreeRejected(struct rte_mbuf** frees, uint16_t* nFree, Packet** npkts, uint64_t mask) {
  uint32_t i = 0;
  while (rte_bsf64_safe(mask, &i)) {
    frees[(*nFree)++] = Packet_ToMbuf(npkts[i]);
    rte_bit_clear(&mask, i);
  }
}

/** @brief InputDemuxes for Interest, Data, Nack. */
typedef InputDemux InputDemuxes[PktMax - 1];

/** @brief Retrieve InputDemux by packet type. */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline InputDemux*
InputDemux_Of(InputDemuxes* demuxes, PktType t) {
  return &((InputDemux*)demuxes)[PktType_ToFull(t) - 1];
}

#endif // NDNDPDK_IFACE_INPUT_DEMUX_H
