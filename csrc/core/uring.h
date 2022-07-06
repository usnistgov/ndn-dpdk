#ifndef NDNDPDK_CORE_URING_H
#define NDNDPDK_CORE_URING_H

/** @file */

#include "common.h"

#include <liburing.h>

/** @brief io_uring and related counters. */
typedef struct Uring
{
  struct io_uring uring;

  uint64_t nAllocErrs;      ///< SQE allocation errors
  uint64_t nSubmitted;      ///< submitted SQEs
  uint64_t nSubmitNonBlock; ///< non-blocking submission batches
  uint64_t nSubmitWait;     ///< waiting submission batches

  uint32_t nQueued;  ///< currently queued but unsubmitted
  uint32_t nPending; ///< currently submitted but uncompleted
} Uring;

/** @brief Initialize io_uring. */
__attribute__((nonnull)) bool
Uring_Init(Uring* ur, uint32_t capacity);

/** @brief Delete io_uring. */
__attribute__((nonnull)) bool
Uring_Free(Uring* ur);

/** @brief Obtain Submission Queue Entry (SQE). */
__attribute__((nonnull)) static __rte_always_inline struct io_uring_sqe*
Uring_GetSqe(Uring* ur)
{
  struct io_uring_sqe* sqe = io_uring_get_sqe(&ur->uring);
  if (unlikely(sqe == NULL)) {
    ++ur->nAllocErrs;
  } else {
    ++ur->nQueued;
  }
  return sqe;
}

__attribute__((nonnull)) void
Uring_Submit_(Uring* ur, uint32_t waitLBound, uint32_t cqeBurst);

/**
 * @brief Submit queued SQEs.
 * @param waitLBound lower bound of @c ur->nPending to use waiting submission.
 * @param cqeBurst number of CQEs to wait for in waiting submission.
 */
__attribute__((nonnull)) static __rte_always_inline void
Uring_Submit(Uring* ur, uint32_t waitLBound, uint32_t cqeBurst)
{
  if (ur->nQueued > 0) {
    Uring_Submit_(ur, waitLBound, cqeBurst);
  }
}

/** @brief Retrieve Completion Queue Entries (CQEs). */
__attribute__((nonnull)) static inline uint32_t
Uring_PeekCqes(Uring* ur, struct io_uring_cqe* cqes[], size_t count)
{
  uint32_t n = io_uring_peek_batch_cqe(&ur->uring, cqes, count);
  ur->nPending -= n;
  return n;
}

/** @brief Release processed CQEs. */
__attribute__((nonnull)) static __rte_always_inline void
Uring_PutCqes(Uring* ur, uint32_t n)
{
  io_uring_cq_advance(&ur->uring, n);
}

#endif // NDNDPDK_CORE_URING_H
