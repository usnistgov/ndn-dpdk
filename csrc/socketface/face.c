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

uint16_t
SocketFace_DgramTxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  SocketFacePriv* priv = Face_GetPriv(face);
  if (unlikely(priv->fd < 0)) {
    goto FREE;
  }

  struct mmsghdr msgs[MaxBurstSize];
  struct iovec iov[LpMaxFragments * MaxBurstSize];
  uint16_t nTx = 0;
  uint16_t nIov = 0;
  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* m = pkts[i];
    if (unlikely(m->nb_segs > LpMaxFragments)) {
      continue;
    }
    msgs[nTx++] = (struct mmsghdr){
      .msg_hdr.msg_iov = &iov[nIov],
      .msg_hdr.msg_iovlen = m->nb_segs,
    };
    for (struct rte_mbuf* seg = m; seg != NULL; seg = seg->next) {
      iov[nIov++] = (struct iovec){
        .iov_base = rte_pktmbuf_mtod(seg, void*),
        .iov_len = seg->data_len,
      };
    }
  }

  int res = sendmmsg(priv->fd, msgs, nTx, MSG_DONTWAIT);
  if (unlikely(res < 0)) {
    SocketFace_HandleError(face, errno);
  }

FREE:;
  rte_pktmbuf_free_bulk(pkts, nPkts);
  return nPkts;
}

STATIC_ASSERT_FUNC_TYPE(Face_TxBurstFunc, SocketFace_DgramTxBurst);
