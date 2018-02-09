#include "eth-rx.h"
#include "eth-face.h"

#include "../../core/logger.h"

INIT_ZF_LOG(EthRx);

static struct rte_mbuf*
EthRx_ProcessFrame(EthFace* face, struct rte_mbuf* pkt)
{
  if (unlikely(pkt->pkt_len < sizeof(struct ether_hdr))) {
    ZF_LOGD("%" PRIu16 " len=%" PRIu32 " no-ether_hdr", pkt->port,
            pkt->pkt_len);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  struct ether_hdr ethBuf;
  const struct ether_hdr* eth =
    rte_pktmbuf_read(pkt, 0, sizeof(ethBuf), &ethBuf);
  if (eth->ether_type != rte_cpu_to_be_16(NDN_ETHERTYPE)) {
    ZF_LOGD("%" PRIu16 " ether_type=%" PRIX16 " not-NDN", pkt->port,
            eth->ether_type);
    rte_pktmbuf_free(pkt);
    return NULL;
  }

  Packet_Adj(pkt, sizeof(struct ether_hdr));
  return pkt;
}

uint16_t
EthRx_RxBurst(EthFace* face, uint16_t queue, struct rte_mbuf** pkts,
              uint16_t nPkts)
{
  uint16_t nInput = rte_eth_rx_burst(face->port, queue, pkts, nPkts);
  uint16_t nOutput = 0;
  for (uint16_t i = 0; i < nInput; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    pkt->port = face->base.id;
    pkt = EthRx_ProcessFrame(face, pkt);
    if (unlikely(pkt == NULL)) {
      continue;
    }
    pkts[nOutput++] = pkt;
  }
  return nOutput;
}
