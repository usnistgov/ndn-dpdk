#ifndef NDN_DPDK_INPUTDEMUX_DEMUX_H
#define NDN_DPDK_INPUTDEMUX_DEMUX_H

/// \file

#include "../iface/face.h"
#include "../ndt/ndt.h"
#include "../pktqueue/queue.h"

typedef struct InputDemuxDest
{
  PktQueue* queue;
  uint64_t nQueued;
  uint64_t nDropped;
} InputDemuxDest;

/** \brief Input packet demuxer for a single packet type.
 */
typedef struct InputDemux InputDemux;

typedef void (*InputDemux_DispatchFunc)(InputDemux* demux,
                                        Packet* npkt,
                                        const Name* name);

void
InputDemux_DispatchDrop(InputDemux* demux, Packet* npkt, const Name* name);

void
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt, const Name* name);

void
InputDemux_DispatchRoundrobinDiv(InputDemux* demux,
                                 Packet* npkt,
                                 const Name* name);

void
InputDemux_DispatchRoundrobinMask(InputDemux* demux,
                                  Packet* npkt,
                                  const Name* name);

void
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt, const Name* name);

void
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt, const Name* name);

#define INPUTDEMUX_DEST_MAX 128

struct InputDemux
{
  InputDemux_DispatchFunc dispatch;
  const Ndt* ndt;
  NdtThread* ndtt;
  uint64_t nDrops;
  struct
  {
    uint32_t i;
    uint32_t n;
  } roundrobin;
  InputDemuxDest dest[INPUTDEMUX_DEST_MAX];
};

static inline void
InputDemux_SetDispatchFunc_(InputDemux* demux, void* f)
{
  demux->dispatch = f;
}

static inline void
InputDemux_SetDispatchRoundrobin_(InputDemux* demux, uint32_t nDest)
{
  if (nDest <= 1) {
    demux->dispatch = InputDemux_DispatchToFirst;
  } else if (RTE_IS_POWER_OF_2(nDest)) {
    demux->roundrobin.n = nDest - 1;
    demux->dispatch = InputDemux_DispatchRoundrobinMask;
  } else {
    demux->roundrobin.n = nDest;
    demux->dispatch = InputDemux_DispatchRoundrobinDiv;
  }
}

static inline void
InputDemux_Dispatch(InputDemux* demux, Packet* npkt, const Name* name)
{
  (*demux->dispatch)(demux, npkt, name);
}

/** \brief Input packet demuxer for all three network layer packet types.
 */
typedef struct InputDemux3
{
  InputDemux interest;
  InputDemux data;
  InputDemux nack;
} InputDemux3;

void
InputDemux3_FaceRx(FaceRxBurst* burst, void* demux0);

#endif // NDN_DPDK_INPUTDEMUX_DEMUX_H
