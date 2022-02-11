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

__attribute__((nonnull)) int
SpdkThread_Run(SpdkThread* th);

__attribute__((nonnull)) int
SpdkThread_Exit(SpdkThread* th);

#endif // NDNDPDK_DPDK_SPDK_THREAD_H
