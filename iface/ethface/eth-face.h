#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H

#include "eth-rx.h"
#include "eth-tx.h"

/// \file

typedef struct EthFace
{
  Face base;
  uint16_t port;
  EthRx rx;
  EthTx tx;
} EthFace;

/** \brief Initialize a face to communicate on Ethernet.
 *  \param[out] face the face
 *  \param port DPDK ethdev port number; must be less than 0x1000
 *  \param indirectMp mempool for indirect mbufs
 *  \param headerMp mempool for Ethernet and NDNLP headers; dataroom must be at least
 *                  EthTxFace_GetHeaderMempoolDataRoom()
 *  \retval 0 success
 *  \retval ENODEV port number is too large
 */
int EthFace_Init(EthFace* face, uint16_t port, struct rte_mempool* indirectMp,
                 struct rte_mempool* headerMp);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
