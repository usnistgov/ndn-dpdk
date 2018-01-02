#include "face.h"

#include "rx-proc.h"

uint16_t
Face_RxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  uint16_t nInputs = (*face->rxBurstOp)(face, pkts, nPkts);
  uint16_t nProcessed = 0;
  for (uint16_t i = 0; i < nInputs; ++i) {
    struct rte_mbuf* processed = RxProc_Input(&face->rx, pkts[i]);
    if (processed != NULL) {
      pkts[nProcessed++] = processed;
    }
  }
  return nProcessed;
}

void
Face_ReadCounters(Face* face, FaceCounters* cnt)
{
  (*face->ops->readCounters)(face, cnt); // TX counters only

  RxProc_ReadCounters(&face->rx, cnt);
}
