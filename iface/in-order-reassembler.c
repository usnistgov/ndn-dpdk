#include "in-order-reassembler.h"

#include "../core/logger.h"

INIT_ZF_LOG(InOrderReassembler);

Packet*
InOrderReassembler_Receive(InOrderReassembler* r, Packet* npkt)
{
  struct rte_mbuf* frame = Packet_ToMbuf(npkt);
  LpL2* lpl2 = &Packet_GetLpHdr(npkt)->l2;
  assert(lpl2->fragCount > 1);
#define PKTDBG(fmt, ...)                                                       \
  ZF_LOGD("%016" PRIX64 ",%" PRIu16 ",%" PRIu16 " " fmt, lpl2->seqNo,          \
          lpl2->fragIndex, lpl2->fragCount, ##__VA_ARGS__)

  if (r->tail == NULL) {
    if (lpl2->fragIndex != 0) {
      PKTDBG("not-first");
      ++r->nOutOfOrder;
      rte_pktmbuf_free(frame);
      return NULL;
    }

    PKTDBG("accepted-first");
    ++r->nAccepted;
    r->head = r->tail = frame;
    r->nextSeqNo = lpl2->seqNo + 1;
    return NULL;
  }

  if (lpl2->seqNo != r->nextSeqNo) {
    PKTDBG("out-of-order, expecting %016" PRIX64, r->nextSeqNo);
    ++r->nOutOfOrder;
    rte_pktmbuf_free(frame);
    rte_pktmbuf_free(r->head);
    r->head = r->tail = NULL;
    return NULL;
  }

  ++r->nAccepted;
  struct rte_mbuf* newTail = rte_pktmbuf_lastseg(frame);
  Packet_Chain(r->head, r->tail, frame);
  r->tail = newTail;
  r->nextSeqNo = lpl2->seqNo + 1;

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
