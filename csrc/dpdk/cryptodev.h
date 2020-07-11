#ifndef NDN_DPDK_DPDK_CRYPTODEV_H
#define NDN_DPDK_DPDK_CRYPTODEV_H

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

static __rte_always_inline struct rte_mempool*
rte_cryptodev_sym_session_pool_create_(const char* name, uint32_t nb_elts, uint32_t elt_size,
                                       uint32_t cache_size, uint16_t priv_size, int socket_id)
{
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"
  return rte_cryptodev_sym_session_pool_create(name, nb_elts, elt_size, cache_size, priv_size,
                                               socket_id);
#pragma GCC diagnostic pop
}

typedef struct CryptoQueuePair
{
  uint8_t dev;
  uint16_t qp;
} CryptoQueuePair;

#endif // NDN_DPDK_DPDK_CRYPTODEV_H
