#include "thread.h"

int
SpdkThread_Run(SpdkThread* th)
{
  while (ThreadStopFlag_ShouldContinue(&th->stop)) {
    spdk_thread_poll(th->spdkTh, 64, 0);
  }
  spdk_thread_exit(th->spdkTh);
  return 0;
}
