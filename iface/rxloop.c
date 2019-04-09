#include "rxloop.h"

RxGroup __theChanRxGroup;

static void
RxLoop_Transfer(RxLoop* rxl, RxGroup* rxg)
{
  uint16_t nRx = (*rxg->rxBurstOp)(
    rxg, FaceRxBurst_GetScratch(rxl->burst), rxl->burst->capacity);
  FaceImpl_RxBurst(rxl->burst, nRx, rxg->rxThread, rxl->cb, rxl->cbarg);
}

void
RxLoop_Run(RxLoop* rxl)
{
  while (ThreadStopFlag_ShouldContinue(&rxl->stop)) {
    rcu_quiescent_state();
    rcu_read_lock();
    RxGroup* rxg;
    cds_hlist_for_each_entry_rcu_2(rxg, &rxl->head, rxlNode)
    {
      RxLoop_Transfer(rxl, rxg);
    }
    rcu_read_unlock();
  }
}
