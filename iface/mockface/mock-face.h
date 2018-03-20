#ifndef NDN_DPDK_IFACE_MOCKFACE_MOCK_FACE_H
#define NDN_DPDK_IFACE_MOCKFACE_MOCK_FACE_H

/// \file

#include "../face.h"

/** \brief A face to communicate on a socket.
 *
 *  MockFace is implemented in Go code. This struct is a proxy to expose MockFace to C code.
 */
typedef struct MockFace
{
  Face base;
} MockFace;

void MockFace_Init(MockFace* face, FaceId id, FaceMempools* mempools);

void MockFace_Rx(MockFace* face, void* cb, void* cbarg, Packet* npkt);

#endif // NDN_DPDK_IFACE_MOCKFACE_MOCK_FACE_H
