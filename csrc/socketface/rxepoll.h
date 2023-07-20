#ifndef NDNDPDK_SOCKETFACE_RXEPOLL_H
#define NDNDPDK_SOCKETFACE_RXEPOLL_H

/** @file */

#include "../iface/rxloop.h"
#include <sys/epoll.h>

/** @brief RX from datagram sockets using epoll. */
typedef struct SocketRxEpoll {
  RxGroup base;
  struct rte_mempool* directMp;
  uint64_t nTruncated;
  int epfd;
  uint16_t msgIndex;
  struct rte_mbuf* mbufs[2 * MaxBurstSize];
  struct iovec iov[2 * MaxBurstSize];
  struct mmsghdr msgs[2 * MaxBurstSize];
} SocketRxEpoll;

__attribute__((nonnull)) void
SocketRxEpoll_PrepareEvent(struct epoll_event* e, FaceID id, int fd);

__attribute__((nonnull)) void
SocketRxEpoll_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx);

#endif // NDNDPDK_SOCKETFACE_RXEPOLL_H
