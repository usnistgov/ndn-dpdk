#include "server.h"
#include "../core/logger.h"
#include "fd.h"
#include "naming.h"

N_LOG_INIT(FileServer);

int
FileServer_Run(FileServer* p)
{
  struct io_uring_params uringParams = { 0 };
  int res = io_uring_queue_init_params(p->uringCapacity, &p->uring, &uringParams);
  if (res < 0) {
    N_LOGE("uring init" N_LOG_ERROR("errno=%d"), -res);
    return 1;
  }
  N_LOGI("uring init sqe=%" PRIu32 " cqe=%" PRIu32 " features=0x%" PRIx32, uringParams.sq_entries,
         uringParams.cq_entries, uringParams.features);
  TAILQ_INIT(&p->fdQ);

  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(p->ctrl, nProcessed)) {
    nProcessed += FileServer_RxBurst(p);
    nProcessed += FileServer_TxBurst(p);
  }

  io_uring_queue_exit(&p->uring);
  FileServerFd_Clear(p);
  return 0;
}
