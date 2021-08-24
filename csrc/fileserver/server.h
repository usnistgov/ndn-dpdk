#ifndef NDNDPDK_FILESERVER_SERVER_H
#define NDNDPDK_FILESERVER_SERVER_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "../ndni/name.h"
#include "enum.h"
#include <liburing.h>

/** @brief FileServer. */
typedef struct FileServer
{
  PktQueue rxQueue;
  struct rte_mempool* payloadMp;
  struct io_uring uring;
  ThreadLoadStat loadStat;
  FaceID face;
  uint16_t segmentLen;
  uint16_t payloadHeadroom;
  ThreadStopFlag stop;
  int dfd[FileServerMaxMounts];
  uint16_t prefixL[FileServerMaxMounts];
  uint8_t prefixV[FileServerMaxMounts * NameMaxLength];
  uint32_t uringCapacity;
} FileServer;

__attribute__((nonnull)) int
FileServer_Run(FileServer* p);

#endif // NDNDPDK_FILESERVER_SERVER_H
