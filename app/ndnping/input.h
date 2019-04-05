#ifndef NDN_DPDK_APP_NDNPING_INPUT_H
#define NDN_DPDK_APP_NDNPING_INPUT_H

/// \file

#include "../../iface/face.h"

typedef struct NdnpingInputEntry
{
  struct rte_ring* clientQueue; ///< queue toward client for Data and Nack
  struct rte_ring* serverQueue; ///< queue toward server for Interest
} NdnpingInputEntry;

/** \brief Input thread.
 */
typedef struct NdnpingInput
{
  uint16_t minFaceId;
  uint16_t maxFaceId;
  NdnpingInputEntry entry[0];
} NdnpingInput;

NdnpingInput*
NdnpingInput_New(uint16_t minFaceId, uint16_t maxFaceId, unsigned numaSocket);

static NdnpingInputEntry*
__NdnpingInput_GetEntry(NdnpingInput* input, uint16_t faceId)
{
  if (faceId >= input->minFaceId && faceId <= input->maxFaceId) {
    return &input->entry[faceId - input->minFaceId];
  }
  return NULL;
}

void
NdnpingInput_FaceRx(FaceRxBurst* burst, void* input0);

#endif // NDN_DPDK_APP_NDNPING_INPUT_H
