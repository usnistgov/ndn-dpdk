#ifndef NDN_DPDK_APP_NDNPING_INPUT_H
#define NDN_DPDK_APP_NDNPING_INPUT_H

/// \file

#include "../../iface/face.h"

typedef struct PingInputEntry
{
  struct rte_ring* clientQueue; ///< queue toward client for Data and Nack
  struct rte_ring* serverQueue; ///< queue toward server for Interest
} PingInputEntry;

/** \brief Input thread.
 */
typedef struct PingInput
{
  uint16_t minFaceId;
  uint16_t maxFaceId;
  PingInputEntry entry[0];
} PingInput;

PingInput*
PingInput_New(uint16_t minFaceId, uint16_t maxFaceId, unsigned numaSocket);

static PingInputEntry*
PingInput_GetEntry(PingInput* input, uint16_t faceId)
{
  if (faceId >= input->minFaceId && faceId <= input->maxFaceId) {
    return &input->entry[faceId - input->minFaceId];
  }
  return NULL;
}

void
PingInput_FaceRx(FaceRxBurst* burst, void* input0);

#endif // NDN_DPDK_APP_NDNPING_INPUT_H
