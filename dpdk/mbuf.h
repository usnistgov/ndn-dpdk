#ifndef NDN_DPDK_DPDK_MBUF_H
#define NDN_DPDK_DPDK_MBUF_H

/// \file

#include "../core/common.h"
#include <rte_mbuf.h>

/** \brief Get private header after struct rte_mbuf.
 *  \param m pointer to struct rte_mbuf
 *  \param T type to cast result to
 *  \param off offset in private headr
 */
#define MbufPriv(m, T, off) ((T)((char*)(m) + sizeof(struct rte_mbuf) + (off)))

#endif // NDN_DPDK_DPDK_MBUF_H