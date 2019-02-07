#ifndef NDN_DPDK_SPDK_THREAD_H
#define NDN_DPDK_SPDK_THREAD_H

/// \file

#include "../dpdk/thread.h"

#include <spdk/thread.h>

typedef struct SpdkThread
{
  struct spdk_thread* spdkTh;
  ThreadStopFlag stop;
} SpdkThread;

static int
SpdkThread_Run(SpdkThread* th)
{
  while (ThreadStopFlag_ShouldContinue(&th->stop)) {
    spdk_thread_poll(th->spdkTh, 64);
  }
  spdk_thread_exit(th->spdkTh);
}

#endif // NDN_DPDK_SPDK_THREAD_H
