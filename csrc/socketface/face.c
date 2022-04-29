#include "face.h"
#include "../core/logger.h"

N_LOG_INIT(SocketFace);

void
SocketFace_HandleError_(Face* face, int err)
{
  SocketFacePriv* priv = Face_GetPriv(face);
  N_LOGW("socket error face=%" PRI_FaceID " fd=%d" N_LOG_ERROR_ERRNO, face->id, priv->fd, err);

  socklen_t len = sizeof(err);
  getsockopt(priv->fd, SOL_SOCKET, SO_ERROR, &err, &len);
}

__attribute__((nonnull)) ssize_t
SocketFace_DgramTx(SocketFacePriv* priv, struct rte_mbuf* m)
{
  if (unlikely(m->nb_segs > LpMaxFragments || priv->fd < 0)) {
    return 0;
  }
  struct iovec iov[LpMaxFragments];
  struct msghdr msg = (struct msghdr){
    .msg_iov = iov,
  };
  for (struct rte_mbuf* seg = m; seg != NULL; seg = seg->next) {
    iov[msg.msg_iovlen++] = (struct iovec){
      .iov_base = rte_pktmbuf_mtod(seg, void*),
      .iov_len = seg->data_len,
    };
  }
  return sendmsg(priv->fd, &msg, MSG_DONTWAIT);
}

uint16_t
SocketFace_DgramTxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  SocketFacePriv* priv = Face_GetPriv(face);

  uint16_t nTx = 0;
  for (; nTx < nPkts; ++nTx) {
    struct rte_mbuf* m = pkts[nTx];
    ssize_t res = SocketFace_DgramTx(priv, m);
    if (unlikely(res < 0)) {
      SocketFace_HandleError(face, errno);
      break;
    }
  }

  rte_pktmbuf_free_bulk(pkts, nTx);
  return nTx;
}

STATIC_ASSERT_FUNC_TYPE(Face_TxBurstFunc, SocketFace_DgramTxBurst);
