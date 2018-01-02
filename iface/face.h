#ifndef NDN_DPDK_IFACE_FACE_H
#define NDN_DPDK_IFACE_FACE_H

#include "rx-proc.h"

/// \file

/** \brief Numeric face identifier.
 */
typedef uint16_t FaceId;

typedef struct Face Face;
typedef struct FaceCounters FaceCounters;

typedef uint16_t (*FaceOps_RxBurst)(Face* face, struct rte_mbuf** pkts,
                                    uint16_t nPkts);
typedef void (*FaceOps_TxBurst)(Face* face, struct rte_mbuf** pkts,
                                uint16_t nPkts);
typedef bool (*FaceOps_Close)(Face* face);
typedef void (*FaceOps_ReadCounters)(Face* face, FaceCounters* cnt);

typedef struct FaceOps
{
  // most frequent ops, rxBurst and txBurst, are placed directly in Face struct
  FaceOps_Close close;
  FaceOps_ReadCounters readCounters;
} FaceOps;

/** \brief Generic network interface.
 */
typedef struct Face
{
  FaceOps_RxBurst rxBurstOp;
  FaceOps_TxBurst txBurstOp;
  const FaceOps* ops;

  RxProc rx;

  FaceId id;
} Face;

static inline bool
Face_Close(Face* face)
{
  return (*face->ops->close)(face);
}

/** \brief Receive and decode a burst of packet.
 *  \param face the face
 *  \param[out] pkts array of network layer packets with PacketPriv
 *  \param nPkts size of \p pkts array
 *  \return number of filled \p pkts elements; if pkts[i] fails decoding or is retained by
 *          reassembler, it will be a null pointer
 */
uint16_t Face_RxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts);

/** \brief Send a burst of packet.
 *  \param face the face
 *  \param pkts array of network layer packets with PacketPriv
 *  \param nPkts size of \p pkt array
 *
 *  This function creates indirect mbufs to reference \p pkts. The caller must free original
 *  \p pkts when no longer needed.
 */
static inline void
Face_TxBurst(Face* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  (*face->txBurstOp)(face, pkts, nPkts);
}

/** \brief Retrieve face counters.
 */
void Face_ReadCounters(Face* face, FaceCounters* cnt);

#endif // NDN_DPDK_IFACE_FACE_H
