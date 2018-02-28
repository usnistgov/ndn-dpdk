#ifndef NDN_DPDK_IFACE_FACE_H
#define NDN_DPDK_IFACE_FACE_H

/// \file

#include "rx-proc.h"
#include "rxburst.h"
#include "tx-proc.h"

typedef struct Face Face;
typedef struct FaceCounters FaceCounters;

/** \brief Receive a burst of L2 frames.
 *  \param[out] pkts L2 frames without Ethernet/etc header;
 *                   callback releases ownership of these frames
 */
typedef uint16_t (*FaceOps_RxBurst)(Face* face, struct rte_mbuf** pkts,
                                    uint16_t nPkts);

/** \brief Transmit a burst of L2 frames.
 *  \param pkts L2 frames with NDNLP header
 *  \return successfully queued packets; callback owns queued frames, but does
 *          not own or release the remaining frames
 */
typedef uint16_t (*FaceOps_TxBurst)(Face* face, struct rte_mbuf** pkts,
                                    uint16_t nPkts);

/** \brief Close a face.
 */
typedef bool (*FaceOps_Close)(Face* face);

/** \brief Determine NumaSocket of a face.
 */
typedef int (*FaceOps_GetNumaSocket)(Face* face);

typedef struct FaceOps
{
  // most frequent ops, rxBurst and txBurst, are placed directly in Face struct
  FaceOps_Close close;
  FaceOps_GetNumaSocket getNumaSocket;
} FaceOps;

/** \brief Generic network interface.
 */
typedef struct Face
{
  FaceOps_RxBurst rxBurstOp;
  FaceOps_TxBurst txBurstOp;
  const FaceOps* ops;

  RxProc rx;
  TxProc tx;

  FaceId id;
} Face;

// ---- functions invoked by user of face system ----

static bool
Face_Close(Face* face)
{
  return (*face->ops->close)(face);
}

static int
Face_GetNumaSocket(Face* face)
{
  return (*face->ops->getNumaSocket)(face);
}

/** \brief Receive and decode a burst of packet.
 *  \param[out] npkts array of L3 packets
 *  \param count size of \p npkts array
 *  \return number of retrieved packets
 *  \deprecated use RxLoop instead
 */
__rte_deprecated uint16_t Face_RxBurst(Face* face, Packet** npkts,
                                       uint16_t count);

/** \brief Send a burst of packet.
 *  \param npkts array of L3 packets; this function does not take ownership
 *  \param count size of \p npkt array
 */
void Face_TxBurst(Face* face, Packet** npkts, uint16_t count);

/** \brief Send a packet.
 *  \param npkt an L3 packet; this function does not take ownership
 */
static void
Face_Tx(Face* face, Packet* npkt)
{
  Face_TxBurst(face, &npkt, 1);
}

/** \brief Retrieve face counters.
 */
void Face_ReadCounters(Face* face, FaceCounters* cnt);

// ---- functions invoked by face implementation ----

typedef struct FaceMempools
{
  /** \brief mempool for indirect mbufs
   */
  struct rte_mempool* indirectMp;

  /** \brief mempool for name linearize upon RX
   *
   *  Dataroom must be at least NAME_MAX_LENGTH.
   */
  struct rte_mempool* nameMp;

  /** \brief mempool for NDNLP headers upon TX
   *
   *  Dataroom must be at least transport-specific-headroom +
   *  EncodeLpHeader_GetHeadroom() + EncodeLpHeader_GetTailroom().
   */
  struct rte_mempool* headerMp;
} FaceMempools;

/** \brief Initialize face RX and TX.
 *  \param mtu transport MTU available for NDNLP packets.
 *  \param headroom headroom before NDNLP header, as required by transport.
 */
void FaceImpl_Init(Face* face, uint16_t mtu, uint16_t headroom,
                   FaceMempools* mempools);

/** \brief Process received frames and invoke upper layer callback.
 *  \param burst FaceRxBurst_GetScratch(burst) shall contain received frames.
 */
void FaceImpl_RxBurst(Face* face, FaceRxBurst* burst, uint16_t nFrames,
                      Face_RxCb cb, void* cbarg);

/** \brief Update counters after a frame is transmitted.
 */
static void
FaceImpl_CountSent(Face* face, struct rte_mbuf* pkt)
{
  TxProc_CountSent(&face->tx, pkt);
}

#endif // NDN_DPDK_IFACE_FACE_H
