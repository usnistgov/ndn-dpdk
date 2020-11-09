#ifndef NDNDPDK_DPDK_CRYPTODEV_H
#define NDNDPDK_DPDK_CRYPTODEV_H

/** @file */

#include "../core/common.h"
#include <rte_cryptodev.h>

static inline enum rte_crypto_op_status
CryptoOp_GetStatus(const struct rte_crypto_op* op)
{
  return op->status;
}

extern struct rte_crypto_sym_xform theSha256DigestXform;

static inline void
CryptoOp_PrepareSha256Digest(struct rte_crypto_op* op, struct rte_mbuf* src, uint32_t offset,
                             uint32_t length, uint8_t* output)
{
  op->sym->m_src = src;
  op->sym->xform = &theSha256DigestXform;
  op->sym->auth.data.offset = offset;
  op->sym->auth.data.length = length;
  op->sym->auth.digest.data = output;
}

typedef struct CryptoQueuePair
{
  uint8_t dev;
  uint16_t qp;
} CryptoQueuePair;

#endif // NDNDPDK_DPDK_CRYPTODEV_H
