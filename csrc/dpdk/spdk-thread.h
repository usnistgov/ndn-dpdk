#ifndef NDNDPDK_DPDK_SPDK_THREAD_H
#define NDNDPDK_DPDK_SPDK_THREAD_H

/** @file */

#include "thread.h"
#include <spdk/thread.h>

typedef struct SpdkThread
{
  ThreadCtrl ctrl;
  struct spdk_thread* spdkTh;
} SpdkThread;

/** @brief Run SPDK thread until stop requested. */
__attribute__((nonnull)) int
SpdkThread_Run(SpdkThread* th);

/** @brief Request SPDK thread to exit and wait until exit. */
__attribute__((nonnull)) int
SpdkThread_Exit(SpdkThread* th);

#endif // NDNDPDK_DPDK_SPDK_THREAD_H
