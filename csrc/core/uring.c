#include "uring.h"
#include "logger.h"

N_LOG_INIT(Uring);

bool
Uring_Init(Uring* ur, uint32_t capacity) {
  *ur = (Uring){0};
  struct io_uring_params params = {0};
  int res = io_uring_queue_init_params(capacity, &ur->uring, &params);
  if (res < 0) {
    N_LOGE("io_uring_queue_init_params error ur=%p" N_LOG_ERROR_ERRNO, ur, res);
    return false;
  }
  N_LOGI("init ur=%p fd=%d sqe=%" PRIu32 " cqe=%" PRIu32 " features=0x%" PRIx32, ur,
         ur->uring.ring_fd, params.sq_entries, params.cq_entries, params.features);
  return true;
}

bool
Uring_Free(Uring* ur) {
  int fd = ur->uring.ring_fd;
  io_uring_queue_exit(&ur->uring);
  N_LOGI("free ur=%p fd=%d alloc-errs=%" PRIu64 " submitted=%" PRIu64 " submit-nonblock=%" PRIu64
         " submit-wait=%" PRIu64,
         ur, fd, ur->nAllocErrs, ur->nSubmitted, ur->nSubmitNonBlock, ur->nSubmitWait);
  return true;
}

void
Uring_Submit_(Uring* ur, uint32_t waitLBound, uint32_t cqeBurst) {
  int res = -1;
  if (unlikely(ur->nPending >= waitLBound)) {
    NDNDPDK_ASSERT(waitLBound >= cqeBurst);
    ++ur->nSubmitWait;
    res = io_uring_submit_and_wait(&ur->uring, cqeBurst);
  } else {
    ++ur->nSubmitNonBlock;
    res = io_uring_submit(&ur->uring);
  }

  if (unlikely(res < 0)) {
    N_LOGE("io_uring_submit error ur=%p" N_LOG_ERROR_ERRNO, ur, res);
  } else {
    ur->nSubmitted += (uint32_t)res;
    ur->nQueued -= (uint32_t)res;
    ur->nPending += (uint32_t)res;
  }
}
