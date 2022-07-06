#ifndef NDNDPDK_FILESERVER_SERVER_H
#define NDNDPDK_FILESERVER_SERVER_H

/** @file */

#include "../core/uring.h"
#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "enum.h"

typedef struct FileServerFd FileServerFd;

typedef struct FileServerCounters
{
  uint64_t reqRead;
  uint64_t reqLs;
  uint64_t reqMetadata;
  uint64_t fdNew;
  uint64_t fdNotFound;
  uint64_t fdUpdateStat;
  uint64_t fdClose;
  uint64_t cqeFail;
} FileServerCounters;

/** @brief File server. */
typedef struct FileServer
{
  Uring ur;
  ThreadCtrl ctrl;
  PktQueue rxQueue;
  FileServerCounters cnt;

  struct rte_mempool* payloadMp;
  struct rte_mempool* fdMp;
  FileServerFd* fdHt;
  struct cds_list_head fdQ;
  TscDuration statValidity;

  uint32_t uringCongestionLbound;
  uint32_t uringWaitLbound;
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
