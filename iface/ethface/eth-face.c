#include "eth-face.h"

static uint16_t
EthFace_RxBurst(Face* faceBase, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFace* face = (EthFace*)faceBase;
  return EthRx_RxBurst(face, 0, pkts, nPkts);
}

static void
EthFace_TxBurst(Face* faceBase, struct rte_mbuf** pkts, uint16_t nPkts)
{
  EthFace* face = (EthFace*)faceBase;
  EthTx_TxBurst(face, &face->tx, pkts, nPkts);
}

static bool
EthFace_Close(Face* faceBase)
{
  EthFace* face = (EthFace*)faceBase;
  EthTx_Close(face, &face->tx);
  return true;
}

static void
EthFace_ReadCounters(Face* faceBase, FaceCounters* cnt)
{
  EthFace* face = (EthFace*)faceBase;
  EthTx_ReadCounters(face, &face->tx, cnt);
}

static const FaceOps ethFaceOps = {
  .close = EthFace_Close,
  .readCounters = EthFace_ReadCounters,
};

int
EthFace_Init(EthFace* face, uint16_t port, struct rte_mempool* indirectMp,
             struct rte_mempool* headerMp)
{
  if (port >= 0x1000) {
    return ENODEV;
  }
  face->base.id = 0x1000 | port;

  face->base.rxBurstOp = EthFace_RxBurst;
  face->base.txBurstOp = EthFace_TxBurst;
  face->base.ops = &ethFaceOps;
  face->port = port;

  face->tx.queue = 0;
  face->tx.indirectMp = indirectMp;
  face->tx.headerMp = headerMp;

  return EthTx_Init(face, &face->tx);
}
