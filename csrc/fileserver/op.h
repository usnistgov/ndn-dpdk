#ifndef NDNDPDK_FILESERVER_OP_H
#define NDNDPDK_FILESERVER_OP_H

/** @file */

#include "../ndni/name.h"
#include "enum.h"

typedef struct FileServerFd FileServerFd;

typedef struct FileServerOpMbufs
{
  struct rte_mbuf* m[FileServerMaxIovecs];
} FileServerOpMbufs;

__attribute__((nonnull)) static inline void
FileServerOpMbufs_Get(FileServerOpMbufs* vector, uint32_t i, struct rte_mbuf** payload,
                      struct rte_mbuf** interest)
{
  *payload = vector->m[i];
  NDNDPDK_ASSERT((*payload)->next != NULL);
  *interest = (*payload)->next;
  (*payload)->next = NULL;
}

__attribute__((nonnull)) static inline void
FileServerOpMbufs_Set(FileServerOpMbufs* vector, uint32_t i, struct rte_mbuf* payload,
                      struct rte_mbuf* interest)
{
  NDNDPDK_ASSERT(payload->next == NULL);
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

__attribute__((nonnull)) static inline void
FileServerOp_Init(FileServerOp* op, FileServerFd* fd, LName prefix, uint64_t segment)
{
  op->fd = fd;
  op->prefix = prefix;
  op->segment = segment;
  op->nIov = 0;
}

__attribute__((nonnull)) static inline bool
FileServerOp_IsContinuous(const FileServerOp* op, LName prefix, uint64_t segment)
{
  return LName_Equal(op->prefix, prefix) && op->segment + op->nIov == segment;
}

#endif // NDNDPDK_FILESERVER_OP_H
