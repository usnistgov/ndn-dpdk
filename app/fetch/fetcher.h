#ifndef NDN_DPDK_APP_FETCH_FETCHER_H
#define NDN_DPDK_APP_FETCH_FETCHER_H

/// \file

#include "../../container/pktqueue/queue.h"
#include "../../dpdk/thread.h"
#include "../../iface/face.h"
#include "logic.h"

/** \brief Fetcher thread.
 */
typedef struct Fetcher
{
  PktQueue rxQueue;
  struct rte_mempool* interestMp;
  FetchLogic logic;
  FaceId face;
  ThreadStopFlag stop;
  NonceGen nonceGen;
  InterestTemplate tpl;
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
