#include "spdk-thread.h"
#include "../core/urcu.h"

int
SpdkThread_Run(SpdkThread* th) {
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

__attribute__((nonnull)) static void
SpdkThread_DoExit(void* th0) {
  SpdkThread* th = (SpdkThread*)th0;
  spdk_thread_exit(th->spdkTh);
}

int
SpdkThread_Exit(SpdkThread* th) {
  spdk_thread_send_msg(th->spdkTh, SpdkThread_DoExit, th);

  spdk_set_thread(th->spdkTh);
  while (!spdk_thread_is_exited(th->spdkTh)) {
    spdk_thread_poll(th->spdkTh, 64, 0);
  }
  spdk_set_thread(NULL);

  return 0;
}
