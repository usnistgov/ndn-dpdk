#ifndef NDNDPDK_SOCKETFACE_FACE_H
#define NDNDPDK_SOCKETFACE_FACE_H

/** @file */

#include "../iface/rxloop.h"

/** @brief Socket face private data. */
typedef struct SocketFacePriv
{
  int fd;
} SocketFacePriv;

__attribute__((nonnull)) void
SocketFace_HandleError_(Face* face, int err);

/**
 * @brief Handle socket error.
 *
 * EAGAIN is ignored.
 */
__attribute__((nonnull)) static inline void
SocketFace_HandleError(Face* face, int err)
{
  if (likely(err == EAGAIN || err == EWOULDBLOCK)) {
    return;
  }
  SocketFace_HandleError_(face, err);
}

/** @brief Transmit a burst of outgoing packets on datagram socket. */
__attribute__((nonnull)) uint16_t
SocketFace_DgramTxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDNDPDK_SOCKETFACE_FACE_H
