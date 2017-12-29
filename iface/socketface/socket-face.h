#ifndef NDN_DPDK_IFACE_SOCKETFACE_SOCKET_FACE_H
#define NDN_DPDK_IFACE_SOCKETFACE_SOCKET_FACE_H

#include "../common.h"
#include "../face.h"

typedef struct SocketFace
{
  Face base;
} SocketFace;

void SocketFace_Init(SocketFace* face, uint16_t id);

#endif // NDN_DPDK_IFACE_SOCKETFACE_SOCKET_FACE_H
