#ifndef NDNDPDK_IFACE_INPUT_DEMUX_H
#define NDNDPDK_IFACE_INPUT_DEMUX_H

/** @file */

#include "../ndt/ndt.h"
#include "faceid.h"
#include "pktqueue.h"

/** @brief Destination of input packet demultiplexer. */
typedef struct InputDemuxDest
{
  PktQueue* queue;
  uint64_t nQueued;
  uint64_t nDropped;
} InputDemuxDest;

typedef enum InputDemuxFunc
{
  InputDemuxFuncDrop,
  InputDemuxFuncToFirst,
  InputDemuxFuncRoundrobinDiv,
  InputDemuxFuncRoundrobinMask,
  InputDemuxFuncGenericHashDiv,
  InputDemuxFuncGenericHashMask,
  InputDemuxFuncByNdt,
  InputDemuxFuncByToken,
} InputDemuxFunc;

/** @brief Input packet demultiplexer for a single packet type. */
typedef struct InputDemux
{
  uint64_t nDrops;
  InputDemuxFunc dispatch;
  union
  {
    struct
    {
      uint32_t i;
      union
      {
        uint32_t n;
        uint32_t mask;
      };
    } div;
    NdtQuerier ndq;
    struct
    {
      uint8_t offset;
    } byToken;
  };
  InputDemuxDest dest[MaxInputDemuxDest];
} InputDemux;

typedef bool (*InputDemux_DispatchFunc)(InputDemux* demux, Packet* npkt);
extern const InputDemux_DispatchFunc InputDemux_DispatchJmp[];

__attribute__((nonnull, returns_nonnull)) NdtQuerier*
InputDemux_SetDispatchByNdt(InputDemux* demux);

__attribute__((nonnull)) void
InputDemux_SetDispatchDiv(InputDemux* demux, uint32_t nDest, bool byGenericHash);

__attribute__((nonnull)) void
InputDemux_SetDispatchByToken(InputDemux* demux, uint8_t offset);

/**
 * @brief Dispatch a packet.
 * @param npkt parsed L3 packet.
 * @param name packet name.
 * @retval true packet is dispatched.
 * @retval false packet is rejected and should be freed by caller.
 */
__attribute__((nonnull, warn_unused_result)) static inline bool
InputDemux_Dispatch(InputDemux* demux, Packet* npkt)
{
  return InputDemux_DispatchJmp[demux->dispatch](demux, npkt);
}

/** @brief InputDemuxes for Interest, Data, Nack. */
typedef InputDemux InputDemuxes[PktMax - 1];

/** @brief Retrieve InputDemux by packet type. */
__attribute__((nonnull, returns_nonnull)) static __rte_always_inline InputDemux*
InputDemux_Of(InputDemuxes* demuxes, PktType t)
{
  return &((InputDemux*)demuxes)[PktType_ToFull(t) - 1];
}

#endif // NDNDPDK_IFACE_INPUT_DEMUX_H
