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
  e->data.u64 = ((uint32_t)fd << (CHAR_BIT * sizeof(FaceID))) | id;
}

__attribute__((nonnull)) static inline void
SocketRxEpoll_HandleEvent(SocketRxEpoll* rxe, RxGroupBurstCtx* ctx, const struct epoll_event* e)
{
  FaceID id = e->data.u64;
  int fd = (uint32_t)(e->data.u64 >> (CHAR_BIT * sizeof(FaceID)));

  for (int i = 0; i < MaxPacketsPerSocket; ++i) {
    NDNDPDK_ASSERT(rxe->nUnusedMbufs > 0);
    NDNDPDK_ASSERT(ctx->nRx < RTE_DIM(ctx->pkts));
    struct rte_mbuf* m = rxe->mbufs[rxe->nUnusedMbufs - 1];
    uint16_t tailroom = rte_pktmbuf_tailroom(m);
    ssize_t res = recv(fd, rte_pktmbuf_mtod(m, uint8_t*), tailroom, MSG_DONTWAIT | MSG_TRUNC);

    if (unlikely(res < 0)) {
      Face* face = Face_Get(id);
      NDNDPDK_ASSERT(face != NULL);
      SocketFace_HandleError(face, errno);
      break;
    }
    if (unlikely(res > (ssize_t)tailroom)) {
      ++rxe->nTruncated;
      continue;
    }

    rte_pktmbuf_append(m, (uint16_t)res);
    Mbuf_SetTimestamp(m, rte_get_tsc_cycles());
    m->port = id;

    ctx->pkts[ctx->nRx++] = m;
    --rxe->nUnusedMbufs;
  }
}

void
SocketRxEpoll_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx)
{
  SocketRxEpoll* rxe = container_of(rxg, SocketRxEpoll, base);

  static_assert(RTE_DIM(rxe->mbufs) == 2 * RTE_DIM(ctx->pkts), "");
  if (unlikely(rxe->nUnusedMbufs < RTE_DIM(rxe->mbufs) / 2)) {
    int res = rte_pktmbuf_alloc_bulk(rxe->directMp, &rxe->mbufs[rxe->nUnusedMbufs],
                                     RTE_DIM(rxe->mbufs) - rxe->nUnusedMbufs);
    if (unlikely(res != 0)) {
      return;
    }
    rxe->nUnusedMbufs = RTE_DIM(rxe->mbufs);
  }

  struct epoll_event events[MaxEvents];
  int res = epoll_wait(rxe->epfd, events, RTE_DIM(events), 0);
  for (int i = 0; i < res; ++i) {
    SocketRxEpoll_HandleEvent(rxe, ctx, &events[i]);
  }
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, SocketRxEpoll_RxBurst);
