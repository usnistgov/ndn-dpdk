#ifndef NDNDPDK_FETCH_FETCHER_H
#define NDNDPDK_FETCH_FETCHER_H

/** @file */

#include "../core/uring.h"
#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "logic.h"

/** @brief Fetch task that fetches from one prefix. */
typedef struct FetchTask
{
  struct cds_hlist_node fthNode; ///< FetchThread.head node
  PktQueue queueD;
  FetchLogic logic;
  uint32_t segmentLen; ///< expected segment length, used for writing to file
  int fd;              ///< if non-negative, write content to file
  uint8_t index;       ///< task slot index, used as PIT token
  int8_t worker;       ///< FetchThread index running this task, -1 if inactive

  /**
   * @brief Name prefix and Interest template.
   *
   * prefixV[prefixL]==TtSegmentNameComponent
   */
  InterestTemplate tpl;
} FetchTask;

/** @brief Fetch thread that runs several fetch procedures. */
typedef struct FetchThread
{
  Uring ur;
  ThreadCtrl ctrl;

  struct rte_mempool* interestMp;
  struct cds_hlist_head tasksHead;
  pcg32_random_t nonceRng;
  uint32_t uringWaitLbound;
  FaceID face;

  uint32_t uringCapacity;
} FetchThread;

__attribute__((nonnull)) int
FetchThread_Run(FetchThread* fth);

#endif // NDNDPDK_FETCH_FETCHER_H
