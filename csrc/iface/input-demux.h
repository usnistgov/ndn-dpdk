#ifndef NDNDPDK_IFACE_INPUT_DEMUX_H
#define NDNDPDK_IFACE_INPUT_DEMUX_H

/** @file */

#include "../ndt/ndt.h"
#include "faceid.h"
#include "pktqueue.h"

typedef struct InputDemuxDest
{
  PktQueue* queue;
  uint64_t nQueued;
  uint64_t nDropped;
} InputDemuxDest;

/** @brief Input packet demultiplexer for a single packet type. */
typedef struct InputDemux InputDemux;

typedef void (*InputDemux_DispatchFunc)(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchDrop(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchRoundrobinDiv(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchRoundrobinMask(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchGenericHashDiv(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchGenericHashMask(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt, const PName* name);

__attribute__((nonnull)) void
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt, const PName* name);

struct InputDemux
{
  InputDemux_DispatchFunc dispatch;
  NdtQuerier* ndq;
  uint64_t nDrops;
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
    struct
    {
      uint8_t offset;
    } byToken;
  };
  InputDemuxDest dest[MaxInputDemuxDest];
};

__attribute__((nonnull)) void
InputDemux_SetDispatchDiv(InputDemux* demux, uint32_t nDest, bool byGenericHash);

__attribute__((nonnull)) void
InputDemux_SetDispatchByToken(InputDemux* demux, uint8_t offset);

__attribute__((nonnull)) static inline void
InputDemux_Dispatch(InputDemux* demux, Packet* npkt, const PName* name)
{
  (*demux->dispatch)(demux, npkt, name);
}

#endif // NDNDPDK_IFACE_INPUT_DEMUX_H
