#ifndef NDN_DPDK_APP_FETCH_FETCHER_H
#define NDN_DPDK_APP_FETCH_FETCHER_H

/// \file

#include "../../dpdk/thread.h"
#include "../../iface/face.h"
#include "../../ndn/encode-interest.h"
#include "logic.h"

/** \brief Fetcher thread.
 */
typedef struct Fetcher
{
  struct rte_ring* rxQueue;
  struct rte_mempool* interestMp;
  FetchLogic logic;
  FaceId face;
  uint16_t interestMbufHeadroom;
  ThreadStopFlag stop;
  NonceGen nonceGen;
  InterestTemplate tpl;
  uint8_t tplPrepareBuffer[64];
  uint8_t suffixBuffer[2 + sizeof(uint64_t)];
  uint8_t prefixBuffer[NAME_MAX_LENGTH];
} Fetcher;

enum
{
  FETCHER_COMPLETED = 0,
  FETCHER_STOPPED = 1,
};

/** \brief Execute fetcher until stopped or fetch completion.
 */
int
Fetcher_Run(Fetcher* fetcher);

#endif // NDN_DPDK_APP_FETCH_FETCHER_H
