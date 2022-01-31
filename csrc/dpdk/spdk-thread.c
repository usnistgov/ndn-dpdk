#include "spdk-thread.h"
#include "../core/urcu.h"

int
SpdkThread_Run(SpdkThread* th)
{
  rcu_register_thread();
  spdk_set_thread(th->spdkTh);
  int work = 0;
  while (ThreadCtrl_Continue(th->ctrl, work)) {
    rcu_quiescent_state();
    work = spdk_thread_poll(th->spdkTh, 64, 0);
  }
  spdk_set_thread(NULL);
  rcu_unregister_thread();
  return 0;
}
