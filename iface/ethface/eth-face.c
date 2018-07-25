#include "eth-face.h"
#include "../../core/logger.h"

INIT_ZF_LOG(EthFace);

// EthFace currently only supports one TX queue,
// so queue number is hardcoded with this macro.
#define QUEUE_0 0

static uint16_t EthFace_TxBurst(Face* face, struct rte_mbuf** pkts,
                                uint16_t nPkts);

static uint16_t
EthFace_GetPort(Face* face)
{
  return face->id & 0x0FFF;
}

static uint16_t
EthFace_TxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                   uint16_t nPkts, void* face0)
{
  Face* face = (Face*)face0;
  assert(EthFace_GetPort(face) == port);
  for (uint16_t i = 0; i < nPkts; ++i) {
    FaceImpl_CountSent(face, pkts[i]);
  }
  return nPkts;
}

bool
EthFace_Init(Face* face, FaceMempools* mempools)
{
  assert(rte_pktmbuf_data_room_size(mempools->headerMp) >=
         EthFace_SizeofTxHeader());
  uint16_t port = EthFace_GetPort(face);
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  uint16_t mtu;
  int res = rte_eth_dev_get_mtu(port, &mtu);
  if (res != 0) {
    assert(res == -ENODEV);
    rte_errno = ENODEV;
    return false;
  }

  face->txBurstOp = EthFace_TxBurst;

  priv->txCallback =
    rte_eth_add_tx_callback(port, QUEUE_0, &EthFace_TxCallback, face);
  if (priv->txCallback == NULL) {
    return false;
  }

  EthDev_GetMacAddr(port, &priv->ethhdr.s_addr);
  const uint8_t dstAddr[] = { NDN_ETHER_MCAST };
  rte_memcpy(&priv->ethhdr.d_addr, dstAddr, sizeof(priv->ethhdr.d_addr));
  priv->ethhdr.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  FaceImpl_Init(face, mtu, sizeof(struct ether_hdr), mempools);
  return true;
}

void
EthFace_Close(Face* face)
{
  uint16_t port = EthFace_GetPort(face);
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);
  rte_eth_remove_tx_callback(port, QUEUE_0, priv->txCallback);
}

static uint16_t
EthFace_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  uint16_t port = EthFace_GetPort(face);
  EthFacePriv* priv = Face_GetPrivT(face, EthFacePriv);

  for (uint16_t i = 0; i < nPkts; ++i) {
    char* room = rte_pktmbuf_prepend(pkts[i], sizeof(priv->ethhdr));
    assert(room != NULL); // enough headroom is required
    rte_memcpy(room, &priv->ethhdr, sizeof(priv->ethhdr));
  }
  return rte_eth_tx_burst(port, QUEUE_0, pkts, nPkts);
}
