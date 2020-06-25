#include "in-order-reassembler.h"

#include "../core/logger.h"

INIT_ZF_LOG(InOrderReassembler);

Packet*
InOrderReassembler_Receive(InOrderReassembler* r, Packet* npkt)
{
  struct rte_mbuf* frame = Packet_ToMbuf(npkt);
  LpL2* lpl2 = &Packet_GetLpHdr(npkt)->l2;
  assert(lpl2->fragCount > 1);
#define PKTDBG(fmt, ...)                                                                           \
  ZF_LOGD("%016" PRIX64 ",%" PRIu16 ",%" PRIu16 " " fmt, lpl2->seqNum, lpl2->fragIndex,            \
          lpl2->fragCount, ##__VA_ARGS__)

  if (lpl2->fragIndex == 0) {
    if (unlikely(r->tail != NULL)) {
      PKTDBG("accepted-first, discard-incomplete");
      ++r->nIncomplete;
      rte_pktmbuf_free(r->head);
    } else {
      PKTDBG("accepted-first");
    }
    ++r->nAccepted;
    r->head = frame;
    r->tail = rte_pktmbuf_lastseg(frame);
    r->nextSeqNo = lpl2->seqNum + 1;
    return NULL;
  }

  if (unlikely(r->tail == NULL || lpl2->seqNum != r->nextSeqNo)) {
    PKTDBG("out-of-order");
    ++r->nOutOfOrder;
    rte_pktmbuf_free(frame);
    return NULL;
  }

  ++r->nAccepted;
  struct rte_mbuf* newTail = rte_pktmbuf_lastseg(frame);
  Packet_Chain(r->head, r->tail, frame);
  r->tail = newTail;
  r->nextSeqNo = lpl2->seqNum + 1;

  if (lpl2->fragIndex + 1 < lpl2->fragCount) {
    PKTDBG("accepted-chained");
    return NULL;
  }

  PKTDBG("accepted-last");
  r->tail = NULL; // indicate the reassembler is idle

  ++r->nDelivered;
  return Packet_FromMbuf(r->head);
#undef PKTDBG
}
