#include "spdk-thread.h"
#include "../core/urcu.h"

int
SpdkThread_Run(SpdkThread* th)
{
  rcu_register_thread();
  spdk_set_thread(th->spdkTh);
  while (ThreadStopFlag_ShouldContinue(&th->stop)) {
    rcu_quiescent_state();
    spdk_thread_poll(th->spdkTh, 64, 0);
  }
  spdk_thread_exit(th->spdkTh);
  rcu_unregister_thread();
  return 0;
}
