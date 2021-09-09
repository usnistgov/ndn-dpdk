#ifndef NDNDPDK_FILESERVER_OP_H
#define NDNDPDK_FILESERVER_OP_H

/** @file */

#include "../ndni/name.h"
#include "enum.h"

/**
 * @brief Indicate whether readv may handle more than one iovecs.
 *
 * This is a compile time constant.
 */
#define FileServer_EnableIovBatching (FileServerMaxIovecs > 1)
static_assert(__builtin_constant_p(FileServer_EnableIovBatching), "");

typedef struct FileServerFd FileServerFd;

typedef struct FileServerOpMbufs
{
  struct rte_mbuf* m[FileServerMaxIovecs];
} FileServerOpMbufs;

__attribute__((nonnull)) static inline void
FileServerOpMbufs_Copy(FileServerOpMbufs* restrict dst, const FileServerOpMbufs* restrict src,
                       uint32_t n)
{
  rte_memcpy(dst->m, src->m, sizeof(dst->m[0]) * n);
}

__attribute__((nonnull)) static inline void
FileServerOpMbufs_Get(FileServerOpMbufs* vector, uint32_t i, struct rte_mbuf** payload,
                      struct rte_mbuf** interest)
{
  *payload = vector->m[i];
  *interest = (*payload)->next;
  (*payload)->next = NULL;
  NULLize(vector->m[i]);
}

__attribute__((nonnull)) static inline void
FileServerOpMbufs_Set(FileServerOpMbufs* vector, uint32_t i, struct rte_mbuf* payload,
                      struct rte_mbuf* interest)
{
  // save interest mbuf in payload->next field to reduce sizeof(FileServerOp)
  // rte_mbuf_sanity_check(payload) would fail until payload->next is cleared
  payload->next = interest;
  vector->m[i] = payload;
}

/** @brief File server readv operation. */
typedef struct FileServerOp
{
  FileServerFd* fd;
  uint64_t segment;
  LName prefix;
  uint32_t nIov;
  struct iovec iov[FileServerMaxIovecs];
  FileServerOpMbufs mbufs;
} FileServerOp;
// FileServerOp is stored in mbuf private area during readv operation.
static_assert(sizeof(FileServerOp) <= sizeof(PacketPriv), "");

/** @brief Initialize @c FileServerOp struct. */
__attribute__((nonnull)) static inline void
FileServerOp_Init(FileServerOp* op, FileServerFd* fd, LName prefix, uint64_t segment)
{
  op->fd = fd;
  op->prefix = prefix;
  op->segment = segment;
  op->nIov = 0;
}

/**
 * @brief Determine whether a new request directly follows the pending operation, and there's room
 * to append iovec.
 * @param op a pending operation.
 * @param prefix mount+name of new request.
 * @param segment segment number of new request.
 */
__attribute__((nonnull)) static inline bool
FileServerOp_Follows(const FileServerOp* op, LName prefix, uint64_t segment)
{
  return op->nIov + 1 < FileServerMaxIovecs && LName_Equal(op->prefix, prefix) &&
         op->segment + op->nIov == segment;
}

__attribute__((nonnull)) static inline void
FileServerOp_AppendIov(FileServerOp* op, struct rte_mbuf* payload, uint16_t payloadLen,
                       struct rte_mbuf* interest)
{
  NDNDPDK_ASSERT(op->nIov < FileServerMaxIovecs);
  op->iov[op->nIov] = (struct iovec){
    .iov_base = rte_pktmbuf_mtod(payload, uint8_t*),
    .iov_len = payloadLen,
  };
  FileServerOpMbufs_Set(&op->mbufs, op->nIov, payload, interest);
  ++op->nIov;
}

#endif // NDNDPDK_FILESERVER_OP_H
