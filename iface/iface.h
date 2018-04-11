#ifndef NDN_DPDK_IFACE_IFACE_H
#define NDN_DPDK_IFACE_IFACE_H

/// \file

#include "face.h"

/** \brief Array of all faces.
 */
extern Face* gFaces[FACEID_MAX];

static Face*
__Face_Get(FaceId faceId)
{
  Face* face = gFaces[faceId];
  assert(face != NULL);
  return face;
}

/** \brief Return whether the face is DOWN.
 */
static int
Face_GetNumaSocket(FaceId faceId)
{
  Face* face = __Face_Get(faceId);
  return (*face->ops->getNumaSocket)(face);
}

/** \brief Return whether the face is DOWN.
 */
static bool
Face_IsDown(FaceId faceId)
{
  // TODO implement
  return false;
}

/** \brief Send a burst of packets.
 *  \param npkts array of L3 packets; face takes ownership
 *  \param count size of \p npkts array
 *
 *  This function is non-thread-safe by default.
 *  Invoke Face.EnableThreadSafeTx in Go API to make this thread-safe.
 */
static void
Face_TxBurst(FaceId faceId, Packet** npkts, uint16_t count)
{
  Face* face = __Face_Get(faceId);
  __Face_TxBurst(face, npkts, count);
}

/** \brief Send a packet.
 *  \param npkt an L3 packet; face takes ownership
 */
static void
Face_Tx(FaceId faceId, Packet* npkt)
{
  Face_TxBurst(faceId, &npkt, 1);
}

#endif // NDN_DPDK_IFACE_IFACE_H
