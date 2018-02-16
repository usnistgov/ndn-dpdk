#include "eth-face.h"

// EthFace currently only supports one RX queue and one TX queue,
// so queue number is hardcoded with this macro.
#define QUEUE_0 0

static uint16_t
EthFace_RxBurst(Face* faceBase, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFace* face = (EthFace*)faceBase;
  return EthRx_RxBurst(face, QUEUE_0, pkts, nPkts);
}

static uint16_t
EthFace_TxBurst(Face* faceBase, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFace* face = (EthFace*)faceBase;
  return EthTx_TxBurst(face, QUEUE_0, pkts, nPkts);
}

static bool
EthFace_Close(Face* faceBase)
{
  EthFace* face = (EthFace*)faceBase;
  EthTx_Close(face, QUEUE_0);
  return true;
}

static int
EthFace_GetNumaSocket(Face* faceBase)
{
  EthFace* face = (EthFace*)faceBase;
  return rte_eth_dev_socket_id(face->port);
}

static const FaceOps ethFaceOps = {
  .close = EthFace_Close,
  .getNumaSocket = EthFace_GetNumaSocket,
};

int
EthFace_Init(EthFace* face, uint16_t port, FaceMempools* mempools)
{
  assert(rte_pktmbuf_data_room_size(mempools->headerMp) >=
         EthTx_GetHeaderMempoolDataRoom());

  if (port >= 0x1000) {
    return ENODEV;
  }

  uint16_t mtu;
  int res = rte_eth_dev_get_mtu(face->port, &mtu);
  if (res != 0) {
    assert(res == -ENODEV);
    return ENODEV;
  }

  face->port = port;
  face->base.id = 0x1000 | port;

  face->base.rxBurstOp = EthFace_RxBurst;
  face->base.txBurstOp = EthFace_TxBurst;
  face->base.ops = &ethFaceOps;

  res = EthTx_Init(face, QUEUE_0);
  if (res != 0) {
    return res;
  }

  FaceImpl_Init(&face->base, mtu, sizeof(struct ether_hdr), mempools);
  return 0;
}
