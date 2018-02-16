#ifndef NDN_DPDK_IFACE_SOCKETFACE_SOCKET_FACE_H
#define NDN_DPDK_IFACE_SOCKETFACE_SOCKET_FACE_H

/// \file

#include "../face.h"

/** \brief A face to communicate on a socket.
 *
 *  SocketFace is implemented in Go code. This struct is a proxy to expose SocketFace to C code.
 */
typedef struct SocketFace
{
  Face base;
} SocketFace;

void SocketFace_Init(SocketFace* face, FaceId id, FaceMempools* mempools);

#endif // NDN_DPDK_IFACE_SOCKETFACE_SOCKET_FACE_H
