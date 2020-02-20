#ifndef NDN_DPDK_APP_FETCH_FETCHER_H
#define NDN_DPDK_APP_FETCH_FETCHER_H

/// \file

#include "../../container/pktqueue/queue.h"
#include "../../dpdk/thread.h"
#include "../../iface/face.h"
#include "logic.h"

typedef struct Fetcher Fetcher;

typedef const InterestTemplate* (*Fetcher_ChooseTpl)(Fetcher* fetcher,
                                                     uint64_t segNum);

#define FETCHER_TEMPLATE_MAX 16

/** \brief Fetcher thread.
 */
struct Fetcher
{
  PktQueue rxQueue;
  struct rte_mempool* interestMp;
  FetchLogic logic;
  NonceGen nonceGen;
  FaceId face;
  ThreadStopFlag stop;
  uint8_t nTpls;
  Fetcher_ChooseTpl chooseTpl;
  InterestTemplate tpl[FETCHER_TEMPLATE_MAX];
};

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
