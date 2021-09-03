#ifndef NDNDPDK_FILESERVER_FD_H
#define NDNDPDK_FILESERVER_FD_H

/** @file */

#include "../ndni/data.h"
#include "../vendor/uthash-handle.h"
#include <fcntl.h>
#include <linux/stat.h>
#include <sys/stat.h>

typedef struct FileServerFd
{
  uint8_t self[0];                     // self reference used in HASH_ADD_BYHASHVALUE
  struct statx st;                     // statx result (.stx_ino is TscTime nextUpdate)
  UT_hash_handle hh;                   // fdHt hashtable handle
  struct rte_mbuf* mbuf;               // mbuf storing this entry
  TAILQ_ENTRY(FileServerFd) queueNode; // fdQ node
  DataEnc_MetaInfoBuffer(15) meta;     // MetaInfo (FinalBlockId only)
  uint64_t lastSeg;                    // last segment number
  int fd;                              // file descriptor
  uint16_t refcnt;                     // number of inflight requests referencing this entry
  uint16_t nameL;                      // mount+path TLV-LENGTH
  uint8_t nameV[NameMaxLength];        // mount+path TLV-VALUE
} FileServerFd;

/** @brief Sentinel value to indicate file not found. */
extern FileServerFd* FileServer_NotFound;

typedef struct FileServer FileServer;

/**
 * @brief Open or reference file descriptor.
 * @param name Interest name.
 * @retval true filename is valid and matches a mount.
 * @retval false filename is invalid or does not match a mount.
 */
__attribute__((nonnull)) FileServerFd*
FileServerFd_Open(FileServer* p, const PName* name, TscTime now);

/** @brief Dereference file descriptor. */
__attribute__((nonnull)) void
FileServerFd_Unref(FileServer* p, FileServerFd* entry);

/** @brief Close all file descriptors. */
__attribute__((nonnull)) void
FileServerFd_Clear(FileServer* p);

enum
{
  FileServerStatxRequired = STATX_TYPE | STATX_MODE | STATX_MTIME | STATX_SIZE,
  FileServerStatxOptional = STATX_ATIME | STATX_CTIME | STATX_BTIME,
};

static __rte_always_inline bool
FileServerFd_HasStatBit(const FileServerFd* entry, uint32_t bit)
{
  return (entry->st.stx_mask & bit) == bit;
}

#endif // NDNDPDK_FILESERVER_FD_H
