#include "in-order-reassembler.h"

#include "../../core/logger.h"

struct rte_mbuf*
InOrderReassembler_Receive(InOrderReassembler* r, struct rte_mbuf* pkt)
{
#define PKTDBG(fmt, ...)                                                       \
  ZF_LOGD("%016" PRIX64 ",%" PRIu16 ",%" PRIu16 " " fmt, lpp->seqNo,           \
          lpp->fragIndex, lpp->fragCount, ##__VA_ARGS__)

  LpPkt* lpp = Packet_GetLpHdr(pkt);
  assert(LpPkt_HasPayload(lpp) & LpPkt_IsFragmented(lpp));

  if (r->tail == NULL) {
    if (lpp->fragIndex != 0) {
      PKTDBG("not-first");
      ++r->nOutOfOrder;
      rte_pktmbuf_free(pkt);
      return NULL;
    }

    PKTDBG("accepted-first");
    ++r->nAccepted;
    r->head = r->tail = pkt;
    r->nextSeqNo = lpp->seqNo + 1;
    return NULL;
  }

  if (lpp->seqNo != r->nextSeqNo) {
    PKTDBG("out-of-order, expecting %016" PRIX64, r->nextSeqNo);
    ++r->nOutOfOrder;
    rte_pktmbuf_free(pkt);
    rte_pktmbuf_free(r->head);
    r->head = r->tail = NULL;
    return NULL;
  }

  ++r->nAccepted;
  // TODO more efficient chaining
  rte_pktmbuf_chain(r->head, pkt);
  r->tail = rte_pktmbuf_lastseg(r->head);
  r->nextSeqNo = lpp->seqNo + 1;

  if (lpp->fragIndex + 1 < lpp->fragCount) {
    PKTDBG("accepted-chained");
    return NULL;
  }

  PKTDBG("accepted-last");
  r->tail = NULL; // indicate the reassembler is idle

  ++r->nDelivered;
  lpp = Packet_GetLpHdr(r->head);
  lpp->fragIndex = lpp->fragCount = 0;
  return r->head;
#undef PKTDBG
}