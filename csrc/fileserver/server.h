#ifndef NDNDPDK_FILESERVER_SERVER_H
#define NDNDPDK_FILESERVER_SERVER_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "enum.h"
#include <liburing.h>

typedef struct FileServerFd FileServerFd;

/** @brief File server. */
typedef struct FileServer
{
  ThreadCtrl ctrl;
  PktQueue rxQueue;

  struct rte_mempool* payloadMp;
  struct io_uring uring;
  FileServerFd* fdHt;
  TAILQ_HEAD(FileServerFdQueue, FileServerFd) fdQ;
  TscDuration statValidity;

  FaceID face;
  uint16_t segmentLen;
  uint16_t payloadHeadroom;
  uint16_t fdQCount;
  uint16_t fdQCapacity;

  int dfd[FileServerMaxMounts];
  int16_t mountPrefixComps[FileServerMaxMounts];
  uint16_t mountPrefixL[FileServerMaxMounts];
  uint8_t mountPrefixV[FileServerMaxMounts * NameMaxLength];

  uint32_t uringCapacity;
  uint32_t nFdHtBuckets;
} FileServer;

__attribute__((nonnull)) uint32_t
FileServer_RxBurst(FileServer* p);

__attribute__((nonnull)) uint32_t
FileServer_TxBurst(FileServer* p);

__attribute__((nonnull)) int
FileServer_Run(FileServer* p);

#endif // NDNDPDK_FILESERVER_SERVER_H
