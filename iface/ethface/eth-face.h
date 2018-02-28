#ifndef NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
#define NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H

/// \file

#include "eth-rx.h"
#include "eth-tx.h"

#define ETHFACE_MAX_RX_COUNT 1
#define ETHFACE_MAX_TX_COUNT 1

/** \brief Ethernet face.
 */
typedef struct EthFace
{
  Face base;
  uint16_t port;
  EthRx rx[ETHFACE_MAX_RX_COUNT];
  EthTx tx[ETHFACE_MAX_TX_COUNT];
} EthFace;

/** \brief Initialize a face to communicate on Ethernet.
 *  \param[out] face the face
 *  \param port DPDK ethdev port number; must be less than 0x1000
 *  \param mempools headerMp must have \p EthTx_GetHeaderMempoolDataRoom() dataroom
 *  \retval 0 success
 *  \retval ENODEV port number is too large
 */
int EthFace_Init(EthFace* face, uint16_t port, FaceMempools* mempools);

/** \brief Continually retrieve packets from an Ethernet face.
 *  \param burstSize how many L2 frames to retrieve in each burst.
 *  \param cb callback after each packet arrival.
 */
void EthFace_RxLoop(EthFace* face, uint16_t burstSize, Face_RxCb cb,
                    void* cbarg);

#endif // NDN_DPDK_IFACE_ETHFACE_ETH_FACE_H
