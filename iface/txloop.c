#include "txloop.h"

#define FACE_TX_LOOP_BURST_SIZE 64

void
FaceTxLoop_Run(FaceTxLoop* txl)
{
  Packet* npkts[FACE_TX_LOOP_BURST_SIZE];

  Face* face = txl->head;
  assert(face->threadSafeTxQueue != NULL);

  while (!txl->stop) {
    uint16_t count = rte_ring_sc_dequeue_burst(
      face->threadSafeTxQueue, (void**)npkts, FACE_TX_LOOP_BURST_SIZE, NULL);
    Face_TxBurst_Nts(face, npkts, count);
  }
}
