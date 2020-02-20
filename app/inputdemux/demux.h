#ifndef NDN_DPDK_APP_INPUTDEMUX_DEMUX_H
#define NDN_DPDK_APP_INPUTDEMUX_DEMUX_H

/// \file

#include "../../container/ndt/ndt.h"
#include "../../container/pktqueue/queue.h"
#include "../../iface/face.h"

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
InputDemux_DispatchByNdt(InputDemux* demux, Packet* npkt, const Name* name);

void
InputDemux_DispatchByToken(InputDemux* demux, Packet* npkt, const Name* name);

void
InputDemux_DispatchToFirst(InputDemux* demux, Packet* npkt, const Name* name);

#define INPUTDEMUX_DEST_MAX 128

struct InputDemux
{
  InputDemux_DispatchFunc dispatch;
  const Ndt* ndt;
  NdtThread* ndtt;
  uint64_t nNoDest;
  InputDemuxDest dest[INPUTDEMUX_DEST_MAX];
};

static inline void
InputDemux_SetDispatchFunc_(InputDemux* demux, void* f)
{
  demux->dispatch = f;
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

#endif // NDN_DPDK_APP_INPUTDEMUX_DEMUX_H
