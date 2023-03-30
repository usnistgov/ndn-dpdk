#include "rxepoll.h"
#include "face.h"

enum
{
  MaxEvents = 4,
  MaxPacketsPerSocket = MaxBurstSize / MaxEvents,
};
static_assert(MaxEvents > 0, "");
static_assert(MaxPacketsPerSocket > 0, "");

void
SocketRxEpoll_PrepareEvent(struct epoll_event* e, FaceID id, int fd)
{
  e->events = EPOLLIN | EPOLLRDHUP | EPOLLERR | EPOLLHUP;
  static_assert(sizeof(int) == sizeof(uint32_t), "");
  static_assert(sizeof(int) + sizeof(FaceID) <= sizeof(uint64_t), "");
  e->data.u64 = ((uint64_t)fd << (CHAR_BIT * sizeof(FaceID))) | id;
}

__attribute__((nonnull)) static inline bool
SocketRxEpoll_RefillBufs(SocketRxEpoll* rxe)
{
  static_assert(RTE_DIM(rxe->mbufs) == RTE_DIM(rxe->msgs), "");
  static_assert(RTE_DIM(rxe->mbufs) == RTE_DIM(rxe->iov), "");

  int res = rte_pktmbuf_alloc_bulk(rxe->directMp, rxe->mbufs, rxe->msgIndex);
  if (unlikely(res != 0)) {
    return false;
  }

  for (uint16_t i = 0; i < rxe->msgIndex; ++i) {
    struct rte_mbuf* m = rxe->mbufs[i];
    rxe->iov[i] = (struct iovec){
      .iov_base = rte_pktmbuf_mtod(m, void*),
      .iov_len = rte_pktmbuf_tailroom(m),
    };
    rxe->msgs[i] = (struct mmsghdr){
      .msg_hdr.msg_iov = &rxe->iov[i],
      .msg_hdr.msg_iovlen = 1,
    };
  }

  rxe->msgIndex = 0;
  return NULL;
}

__attribute__((nonnull)) static inline void
SocketRxEpoll_HandleEvent(SocketRxEpoll* rxe, RxGroupBurstCtx* ctx, const struct epoll_event* e)
{
  FaceID id = e->data.u64;
  int fd = (uint32_t)(e->data.u64 >> (CHAR_BIT * sizeof(FaceID)));

  NDNDPDK_ASSERT((size_t)(rxe->msgIndex + MaxPacketsPerSocket) <= RTE_DIM(rxe->msgs));
  int res = recvmmsg(fd, &rxe->msgs[rxe->msgIndex], MaxPacketsPerSocket, MSG_DONTWAIT, NULL);
  if (unlikely(res < 0)) {
    Face* face = Face_Get(id);
    NDNDPDK_ASSERT(face != NULL);
    SocketFace_HandleError(face, errno);
    return;
  }

  TscTime now = rte_get_tsc_cycles();
  for (uint16_t last = rxe->msgIndex + res; rxe->msgIndex < last; ++rxe->msgIndex) {
    struct rte_mbuf* m = rxe->mbufs[rxe->msgIndex];
    rte_pktmbuf_append(m, rxe->msgs[rxe->msgIndex].msg_len);
    Mbuf_SetTimestamp(m, now);
    m->port = id;
    ctx->pkts[ctx->nRx++] = m;
  }
}

void
SocketRxEpoll_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  SocketRxEpoll* rxe = container_of(rxg, SocketRxEpoll, base);

  static_assert(RTE_DIM(rxe->mbufs) == 2 * RTE_DIM(ctx->pkts), "");
  if (unlikely(rxe->msgIndex > RTE_DIM(ctx->pkts)) && unlikely(!SocketRxEpoll_RefillBufs(rxe))) {
    return;
  }

  struct epoll_event events[MaxEvents];
  int res = epoll_wait(rxe->epfd, events, RTE_DIM(events), 0);
  for (int i = 0; i < res; ++i) {
    SocketRxEpoll_HandleEvent(rxe, ctx, &events[i]);
  }
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, SocketRxEpoll_RxBurst);
