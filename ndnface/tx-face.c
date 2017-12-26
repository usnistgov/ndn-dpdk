#include "tx-face.h"
#include "../core/logger.h"

static uint16_t
TxFace_TxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                  uint16_t nPkts, void* face0)
{
  TxFace* face = (TxFace*)(face0);

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    ++face->nPkts[Packet_GetNdnPktType(pkt)];
    face->nOctets += pkt->pkt_len;
  }

  return nPkts;
}

bool
TxFace_Init(TxFace* face)
{
  assert(face->indirectMp != NULL);
  assert(face->headerMp != NULL);
  assert(rte_pktmbuf_data_room_size(face->headerMp) >=
         TxFace_GetHeaderMempoolDataRoom());

  int res = rte_eth_dev_get_mtu(face->port, &face->mtu);
  if (res != 0) {
    return false;
  }

  rte_eth_macaddr_get(face->port, &face->ethhdr.s_addr);
  memset(&face->ethhdr.d_addr, 0xFF, sizeof(face->ethhdr.d_addr));
  face->ethhdr.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  face->__txCallback =
    rte_eth_add_tx_callback(face->port, face->queue, &TxFace_TxCallback, face);
  if (face->__txCallback == NULL) {
    return false;
  }

  return true;
}

void
TxFace_Close(TxFace* face)
{
  rte_eth_remove_tx_callback(face->port, face->queue, face->__txCallback);
  face->__txCallback = NULL;
}

static inline void
TxFace_SendFrames(TxFace* face, struct rte_mbuf** frames, uint16_t nFrames)
{
  ++face->nBursts;

  uint16_t nSent = rte_eth_tx_burst(face->port, face->queue, frames, nFrames);
  if (nSent == nFrames) {
    return;
  }

  ++face->nPartialBursts;
  face->nZeroBursts += (nSent == 0);
  for (uint16_t i = nSent; i < nFrames; ++i) {
    rte_pktmbuf_free(frames[i]);
  }
}

void
TxFace_TxBurst(TxFace* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  assert(face->mtu > 0);

  static const int MAX_FRAMES = 64;
  struct rte_mbuf* frames[MAX_FRAMES];
  int nFrames = 0;

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* payload = rte_pktmbuf_clone(pkts[i], face->indirectMp);
    if (unlikely(payload == NULL)) {
      ++face->nAllocFails;
      break;
    }

    // TODO create multiple frames if fragmentation is needed
    struct rte_mbuf* frame = rte_pktmbuf_alloc(face->headerMp);
    if (unlikely(payload == NULL)) {
      ++face->nAllocFails;
      rte_pktmbuf_free(payload);
      break;
    }
    frame->data_off = sizeof(struct ether_hdr);

    struct ether_hdr* eth =
      (struct ether_hdr*)rte_pktmbuf_prepend(frame, sizeof(struct ether_hdr));
    assert(eth != NULL);
    memcpy(eth, &face->ethhdr, sizeof(*eth));

    // TODO fragmentation
    Packet_SetNdnPktType(frame, Packet_GetNdnPktType(payload));
    rte_pktmbuf_chain(frame, payload);

    frames[nFrames++] = frame;

    if (unlikely(nFrames == MAX_FRAMES)) {
      TxFace_SendFrames(face, frames, nFrames);
      nFrames = 0;
    }
  }

  if (likely(nFrames > 0)) {
    TxFace_SendFrames(face, frames, nFrames);
  }
}