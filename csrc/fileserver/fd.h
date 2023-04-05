#ifndef NDNDPDK_FILESERVER_FD_H
#define NDNDPDK_FILESERVER_FD_H

/** @file */

#include "../ndni/data.h"
#include "../ndni/tlv-encoder.h"
#include "../vendor/uthash-handle.h"
#include "enum.h"
#include <fcntl.h>
#include <linux/stat.h>
#include <sys/stat.h>

/**
 * @brief Convert timestamp to uint64_t nanoseconds.
 * @param t struct statx_timestamp or struct timespec.
 */
#define FileServerFd_StatTime(t) ((uint64_t)(t).tv_sec * SPDK_SEC_TO_NSEC + (uint64_t)(t).tv_nsec)

/** @brief File descriptor related information in the file server. */
typedef struct FileServerFd
{
  RTE_MARKER self;                 ///< self reference used in HASH_ADD_BYHASHVALUE
  struct statx st;                 ///< statx result
  UT_hash_handle hh;               ///< fdHt hashtable handle
  struct cds_list_head queueNode;  ///< fdQ node
  uint8_t meta[16];                ///< MetaInfo (FinalBlockId only)
  uint64_t version;                ///< version number
  uint64_t lastSeg;                ///< last segment number
  int fd;                          ///< file descriptor
  uint32_t lsL;                    ///< directory listing length (UINT32_MAX means invalid)
  uint16_t refcnt;                 ///< number of inflight SQEs referencing this entry
  uint16_t prefixL;                ///< mount+path TLV-LENGTH
  uint16_t versionedL;             ///< mount+path+[32=ls]+version TLV-LENGTH
  uint8_t metadataL;               ///< metadata length excluding Name (0 means invalid)
  uint8_t nameV[NameMaxLength];    ///< mount+path+[32=ls]+version TLV-VALUE
  char lsV[FileServerMaxLsResult]; ///< directory listing value
  uint8_t metadataV[FileServerEstimatedMetadataSize]; ///< metadata value excluding Name
} FileServerFd;
static_assert(FileServerEstimatedMetadataSize <= UINT8_MAX, "");

/** @brief Sentinel value to indicate file not found. */
extern FileServerFd* FileServer_NotFound;

typedef struct FileServer FileServer;

/**
 * @brief Open file descriptor or increment reference count.
 * @param name Interest name.
 * @retval NULL filename is invalid or does not match a mount.
 * @retval FileServer_NotFound file does not exist; do not Unref this value.
 */
__attribute__((nonnull)) FileServerFd*
FileServerFd_Open(FileServer* p, const PName* name, TscTime now);

/** @brief Decrement file descriptor reference count. */
__attribute__((nonnull)) void
FileServerFd_Unref(FileServer* p, FileServerFd* entry);

/** @brief Close all file descriptors. */
__attribute__((nonnull)) void
FileServerFd_Clear(FileServer* p);

/** @brief Determine whether this entry refers to a regular file. */
static __rte_always_inline bool
FileServerFd_IsFile(const FileServerFd* entry)
{
  return S_ISREG(entry->st.stx_mode);
}

/** @brief Determine whether this entry refers to a directory. */
static __rte_always_inline bool
FileServerFd_IsDir(const FileServerFd* entry)
{
  return S_ISDIR(entry->st.stx_mode);
}

__attribute__((nonnull)) static inline uint32_t
FileServerFd_SizeofMetadata_(FileServerFd* entry)
{
  return TlvEncoder_SizeofVarNum(TtName) + TlvEncoder_SizeofVarNum(entry->versionedL) +
         entry->versionedL + entry->metadataL;
}

__attribute__((nonnull)) uint32_t
FileServerFd_PrepareMetadata_(FileServer* p, FileServerFd* entry);

/**
 * @brief Prepare metadata packet.
 * @param entry a valid FileServerFd entry.
 * @return metadata Content TLV-LENGTH.
 */
__attribute__((nonnull)) static inline uint32_t
FileServerFd_PrepareMetadata(FileServer* p, FileServerFd* entry)
{
  if (entry->metadataL != 0) {
    return FileServerFd_SizeofMetadata_(entry);
  }
  return FileServerFd_PrepareMetadata_(p, entry);
}

/**
 * @brief Write metadata packet.
 * @param entry a valid FileServerFd entry.
 * @param iov Content iov, must match the length from @c FileServerFd_PrepareMetadata .
 */
__attribute__((nonnull)) void
FileServerFd_WriteMetadata(FileServerFd* entry, struct iovec* iov, int iovcnt);

/**
 * @brief Populate directory listing.
 * @param entry a valid FileServerFd entry representing a directory.
 * @post @c entry->lsV[:entry->lsL] contains 32=ls payload.
 * @post @c entry->lastSeg and @c entry->meta FinalBlock reflects @c entry->lsL size.
 */
__attribute__((nonnull)) bool
FileServerFd_GenerateLs(FileServer* p, FileServerFd* entry);

#endif // NDNDPDK_FILESERVER_FD_H
