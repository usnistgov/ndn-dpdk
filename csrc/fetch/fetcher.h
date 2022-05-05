#ifndef NDNDPDK_FETCH_FETCHER_H
#define NDNDPDK_FETCH_FETCHER_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "logic.h"

/** @brief Fetch procedure that fetches from one prefix. */
typedef struct FetchProc
{
  struct cds_hlist_node fthNode;
  PktQueue rxQueue;
  FetchLogic logic;
  uint8_t pitToken;
  InterestTemplate tpl;
} FetchProc;

/** @brief Fetch thread that runs several fetch procedures. */
typedef struct FetchThread
{
  ThreadCtrl ctrl;
  struct rte_mempool* interestMp;
  struct cds_hlist_head head;
  pcg32_random_t nonceRng;
  FaceID face;
} FetchThread;

__attribute__((nonnull)) int
FetchThread_Run(FetchThread* fth);

#endif // NDNDPDK_FETCH_FETCHER_H
