#ifndef NDN_DPDK_DPDK_SPDK_THREAD_H
#define NDN_DPDK_DPDK_SPDK_THREAD_H

/// \file

#include "thread.h"
#include <spdk/thread.h>

typedef struct SpdkThread
{
  struct spdk_thread* spdkTh;
  ThreadStopFlag stop;
} SpdkThread;

int
SpdkThread_Run(SpdkThread* th);

#endif // NDN_DPDK_DPDK_SPDK_THREAD_H
