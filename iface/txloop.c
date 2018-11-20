#include "txloop.h"

#define FACE_TX_LOOP_BURST_SIZE 64

static void
TxLoop_Transfer(Face* face)
{
  Packet* npkts[FACE_TX_LOOP_BURST_SIZE];
  uint16_t count = rte_ring_sc_dequeue_burst(
    face->threadSafeTxQueue, (void**)npkts, FACE_TX_LOOP_BURST_SIZE, NULL);
  Face_TxBurst_Nts(face, npkts, count);
}

void
MultiTxLoop_Run(MultiTxLoop* txl)
{
  while (!txl->stop) {
    rcu_quiescent_state();
    rcu_read_lock();
    Face* face;
    cds_hlist_for_each_entry_rcu_2(face, &txl->head, threadSafeTxNode)
    {
      TxLoop_Transfer(face);
    }
    rcu_read_unlock();
  }
}
