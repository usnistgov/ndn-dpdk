#ifndef NDNDPDK_FILESERVER_SERVER_H
#define NDNDPDK_FILESERVER_SERVER_H

/** @file */

#include "../dpdk/thread.h"
#include "../iface/face.h"
#include "../iface/pktqueue.h"
#include "../ndni/name.h"
#include "../vendor/uthash-handle.h"
#include "enum.h"
#include <liburing.h>
#include <linux/stat.h>

typedef struct FileServerFd
{
  uint8_t self[0];                     // self reference for HASH_ADD_BYHASHVALUE
  UT_hash_handle hh;                   // fdHt hashtable handle
  TAILQ_ENTRY(FileServerFd) queueNode; // fdQ node
  struct rte_mbuf* mbuf;               // mbuf storing this entry
  int fd;                              // file descriptor
  uint16_t refcnt;                     // number of inflight requests referencing this entry
  uint16_t nameL;                      // mount+path TLV-LENGTH
  uint8_t nameV[NameMaxLength];        // mount+path TLV-VALUE
  uint64_t lastSeg;                    // last segment number
  uint16_t lastLen;                    // last segment length
  MetaInfoBuffer meta;                 // MetaInfo with FinalBlock
  struct statx st;                     // statx result
} FileServerFd;

/** @brief Sentinel value to indicate file not found. */
extern FileServerFd* FileServer_NotFound;

extern const unsigned FileServer_StatxFlags_;

/** @brief File server. */
typedef struct FileServer
{
  ThreadCtrl ctrl;
  PktQueue rxQueue;
  struct rte_mempool* payloadMp;
  struct io_uring uring;
  FileServerFd* fdHt;
  TAILQ_HEAD(FileServerFdQueue, FileServerFd) fdQ;
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

/**
 * @brief Open or reference file descriptor.
 * @param name Interest name.
 * @retval true filename is valid and matches a mount.
 * @retval false filename is invalid or does not match a mount.
 */
__attribute__((nonnull)) FileServerFd*
FileServer_FdOpen(FileServer* p, const PName* name);

/** @brief Dereference file descriptor. */
__attribute__((nonnull)) void
FileServer_FdUnref(FileServer* p, FileServerFd* entry);

/** @brief Close all file descriptors. */
__attribute__((nonnull)) void
FileServer_FdClear(FileServer* p);

__attribute__((nonnull)) int
FileServer_Run(FileServer* p);

#endif // NDNDPDK_FILESERVER_SERVER_H
