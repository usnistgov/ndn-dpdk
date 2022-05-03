#ifndef NDNDPDK_DPDK_CRYPTODEV_H
#define NDNDPDK_DPDK_CRYPTODEV_H

/** @file */

#include "../core/common.h"
#include <rte_cryptodev.h>

__attribute__((nonnull)) struct rte_cryptodev_sym_session*
CryptoDev_NewSha256DigestSession(struct rte_mempool* mp, uint8_t dev);

/** @brief Identify a crypto queue pair. */
typedef struct CryptoQueuePair
{
  struct rte_cryptodev_sym_session* sha256;
  uint8_t dev;
  uint16_t qp;
} CryptoQueuePair;

/**
 * @brief Reset and prepare a crypto operation for SHA256 digest.
 * @param cqp crypto queue pair where this operation is to be submitted.
 * @param[inout] op crypto operation, must have room for rte_crypto_sym_op.
 * @param m input mbuf.
 * @param offset offset within input mbuf.
 * @param length length of input.
 * @param output output buffer, must have 32 octets.
 */
__attribute__((nonnull)) static inline void
CryptoQueuePair_PrepareSha256(CryptoQueuePair* cqp, struct rte_crypto_op* op, struct rte_mbuf* m,
                              uint32_t offset, uint32_t length, uint8_t* output)
{
  __rte_crypto_op_reset(op, RTE_CRYPTO_OP_TYPE_SYMMETRIC);
  op->sym->m_src = m;
  op->sym->auth.data.offset = offset;
  op->sym->auth.data.length = length;
  op->sym->auth.digest.data = output;
  int res = rte_crypto_op_attach_sym_session(op, cqp->sha256);
  NDNDPDK_ASSERT(res == 0);
}

#endif // NDNDPDK_DPDK_CRYPTODEV_H
