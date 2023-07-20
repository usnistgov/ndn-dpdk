#include "server.h"
#include "fd.h"
#include "naming.h"

int
FileServer_Run(FileServer* p) {
  bool ok = Uring_Init(&p->ur, p->uringCapacity);
  if (unlikely(!ok)) {
    return 1;
  }
  CDS_INIT_LIST_HEAD(&p->fdQ);

  uint32_t nProcessed = 0;
  while (ThreadCtrl_Continue(p->ctrl, nProcessed)) {
    nProcessed += FileServer_RxBurst(p);
    nProcessed += FileServer_TxBurst(p);
  }

  Uring_Free(&p->ur);
  FileServerFd_Clear(p);
  return 0;
}
