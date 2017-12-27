#ifndef NDN_DPDK_DPDK_MBUF_LOC_H
#define NDN_DPDK_DPDK_MBUF_LOC_H

/// \file

#include "mbuf.h"
#include <rte_errno.h>
#include <rte_memcpy.h>
#include <rte_prefetch.h>

/** \brief Iterator within a packet.
 *
 *  This struct contains an octet position within a multi-segment packet.
 *  It can optionally carry a boundary so that the iterator cannot be advanced past this limit.
 */
typedef struct MbufLoc
{
  const struct rte_mbuf* m; ///< current segment
  uint32_t rem;             ///< remaining octets before reaching boundary
  uint16_t off;             ///< offset within current segment
} MbufLoc;

/** \brief Initialize a MbufLoc to the beginning of a packet.
 */
static inline void
MbufLoc_Init(MbufLoc* ml, const struct rte_mbuf* pkt)
{
  ml->m = pkt;
  ml->off = 0;
  ml->rem = pkt->pkt_len;

  while (ml->m != NULL && ml->m->data_len == 0) {
    ml->m = ml->m->next;
  }
}

/** \brief Copy MbufLoc \p src to \p dst.
 */
static inline void
MbufLoc_Copy(MbufLoc* dst, const MbufLoc* src)
{
  rte_memcpy(dst, src, sizeof(*dst));
}

/** \brief Test if the iterator points past the end of packet or boundary.
 */
static inline bool
MbufLoc_IsEnd(const MbufLoc* ml)
{
  return ml->m == NULL || ml->rem == 0;
}

typedef void (*MbufLoc_AdvanceCb)(void* arg, const struct rte_mbuf* m,
                                  uint16_t off, uint16_t len);

static inline uint32_t
__MbufLoc_AdvanceWithCb(MbufLoc* ml, uint32_t n, MbufLoc_AdvanceCb cb,
                        void* cbarg)
{
  assert(n <= ml->rem);

  if (unlikely(MbufLoc_IsEnd(ml))) {
    return 0;
  }

  uint32_t dist = 0;
  while (unlikely(ml->m != NULL && ml->off + n >= ml->m->data_len)) {
    uint16_t len = ml->m->data_len - ml->off;
    if (len > 0 && cb != NULL) {
      (*cb)(cbarg, ml->m, ml->off, len);
    }
    dist += len;
    n -= len;
    ml->m = ml->m->next;
    ml->off = 0;
  }

  if (ml->m != NULL) {
    if (cb != NULL) {
      (*cb)(cbarg, ml->m, ml->off, n);
    }
    dist += n;
    ml->off += n;
  }

  ml->rem -= dist;
  return dist;
}

/** \brief Advance the position by n octets.
 *  \return Actually advanced distance.
 */
static inline uint32_t
MbufLoc_Advance(MbufLoc* ml, uint32_t n)
{
  if (n > ml->rem) {
    n = ml->rem;
  }
  return __MbufLoc_AdvanceWithCb(ml, n, NULL, NULL);
}

/** \brief Determine the distance in octets from a to b.
 *
 *  If MbufLoc_Diff(a, b) == n and n >= 0, it implies MbufLoc_Advance(a, n) equals b.
 *  If MbufLoc_Diff(a, b) == n and n <= 0, it implies MbufLoc_Advance(b, -n) equals a.
 *  Behavior is undefined if a and b do not point to the same packet.
 *  This function does not honor the iterator boundary.
 */
ptrdiff_t MbufLoc_Diff(const MbufLoc* a, const MbufLoc* b);

/** \brief Determine the distance in octets from a to b.
 *
 *  This is faster than MbufLoc_Diff, but it requires \p a to be a copy of \p b at an earlier time.
 *  \code
 *  // initialize b
 *  MbufLoc a;
 *  MbufLoc_Copy(&a, &b);
 *  // advance or read from b
 *  uint32_t diff = MbufLoc_FastDiff(a, b);
 *  \endcode
 */
static inline uint32_t
MbufLoc_FastDiff(const MbufLoc* a, const MbufLoc* b)
{
  return a->rem - b->rem;
}

typedef struct __MbufLoc_MakeIndirectCtx
{
  struct rte_mempool* mp;
  struct rte_mbuf* head;
  struct rte_mbuf* tail;
} __MbufLoc_MakeIndirectCtx;

void __MbufLoc_MakeIndirectCb(void* arg, const struct rte_mbuf* m, uint16_t off,
                              uint16_t len);

/** \brief Advance the position by n octets, and clone the range into indirect mbufs.
 *  \return head of indirect mbufs
 *  \retval NULL remaining range is less than \p n (rte_errno=ERANGE), or
                 allocation failure (rte_errno=ENOENT)
 */
static inline struct rte_mbuf*
MbufLoc_MakeIndirect(MbufLoc* ml, uint32_t n, struct rte_mempool* mp)
{
  if (unlikely(MbufLoc_IsEnd(ml) || n > ml->rem)) {
    rte_errno = ERANGE;
    return NULL;
  }

  __MbufLoc_MakeIndirectCtx ctx;
  ctx.mp = mp;
  ctx.head = ctx.tail = NULL;

  __MbufLoc_AdvanceWithCb(ml, n, __MbufLoc_MakeIndirectCb, &ctx);

  if (unlikely(ctx.mp == NULL)) {
    rte_errno = ENOENT;
    if (ctx.head != NULL) {
      rte_pktmbuf_free(ctx.head);
    }
  }

  return ctx.head;
}

void __MbufLoc_ReadCb(void* arg, const struct rte_mbuf* m, uint16_t off,
                      uint16_t len);

/** \brief Read next n octets, and advance the position.
 *  \param buf a buffer to copy octets into, used only if crossing segment boundary.
 *  \param n requested length
 *  \param[out] nRead actual length before reaching end or boundary
 *  \return pointer to in-segment data or the buffer.
 */
static inline const uint8_t*
MbufLoc_Read(MbufLoc* ml, void* buf, uint32_t n, uint32_t* nRead)
{
  if (unlikely(MbufLoc_IsEnd(ml))) {
    *nRead = 0;
    return buf;
  }

  if (n > ml->rem) {
    n = ml->rem;
  }

  if (unlikely(ml->off + n >= ml->m->data_len)) {
    uint8_t* output = (uint8_t*)buf;
    *nRead = __MbufLoc_AdvanceWithCb(ml, n, __MbufLoc_ReadCb, &output);
    return (const uint8_t*)buf;
  }

  *nRead = n;
  ml->off += (uint16_t)n;
  ml->rem -= n;
  return rte_pktmbuf_mtod_offset(ml->m, uint8_t*, ml->off);
}

/** \brief Copy next n octets, and advance the position.
 *  \return number of octets copied.
 */
static inline uint32_t
MbufLoc_ReadTo(MbufLoc* ml, void* output, uint32_t n)
{
  uint32_t nRead;
  const uint8_t* data = MbufLoc_Read(ml, output, n, &nRead);

  if (likely(data != output)) {
    rte_memcpy(output, data, nRead);
  }
  return nRead;
}

static inline bool
MbufLoc_ReadU8(MbufLoc* ml, uint8_t* output)
{
  return sizeof(uint8_t) == MbufLoc_ReadTo(ml, output, sizeof(uint8_t));
}

static inline bool
MbufLoc_ReadU16(MbufLoc* ml, uint16_t* output)
{
  return sizeof(uint16_t) == MbufLoc_ReadTo(ml, output, sizeof(uint16_t));
}

static inline bool
MbufLoc_ReadU32(MbufLoc* ml, uint32_t* output)
{
  return sizeof(uint32_t) == MbufLoc_ReadTo(ml, output, sizeof(uint32_t));
}

static inline bool
MbufLoc_ReadU64(MbufLoc* ml, uint64_t* output)
{
  return sizeof(uint64_t) == MbufLoc_ReadTo(ml, output, sizeof(uint64_t));
}

/** \brief Read the next octet without advancing the iterator.
 *  \return the next octet
 *  \retval -1 iterator is at the end
 */
static inline int
MbufLoc_PeekOctet(const MbufLoc* ml)
{
  if (unlikely(MbufLoc_IsEnd(ml))) {
    return -1;
  }

  uint8_t* data = rte_pktmbuf_mtod_offset(ml->m, uint8_t*, ml->off);
  return *data;
}

#endif // NDN_DPDK_DPDK_MBUF_LOC_H