#include "in-order-reassembler.h"
#include "packet.h"

struct rte_mbuf*
InOrderReassembler_Receive(InOrderReassembler* r, struct rte_mbuf* pkt)
{
  LpPkt* lpp = Packet_GetLpHdr(pkt);
  assert(LpPkt_HasPayload(lpp) & LpPkt_IsFragmented(lpp));

  if (r->tail == NULL) {
    if (lpp->fragIndex != 0) {
      ++r->nOutOfOrder;
      rte_pktmbuf_free(pkt);
      return NULL;
    }

    ++r->nAccepted;
    r->head = r->tail = pkt;
    r->nextSeqNo = lpp->seqNo + 1;
    return NULL;
  }

  if (lpp->seqNo != r->nextSeqNo) {
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
    return NULL;
  }

  ++r->nDelivered;
  r->tail = NULL; // indicate the reassembler is idle
  return r->head;
}