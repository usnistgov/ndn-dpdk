#ifndef NDNDPDK_FILESERVER_FD_H
#define NDNDPDK_FILESERVER_FD_H

/** @file */

#include "../ndni/data.h"
#include "../vendor/uthash-handle.h"
#include <fcntl.h>
#include <linux/stat.h>
#include <sys/stat.h>

/** @brief File descriptor related information in the file server. */
typedef struct FileServerFd
{
  RTE_MARKER self;                 ///< self reference used in HASH_ADD_BYHASHVALUE
  struct statx st;                 ///< statx result (.stx_ino is TscTime nextUpdate)
  UT_hash_handle hh;               ///< fdHt hashtable handle
  struct rte_mbuf* mbuf;           ///< mbuf storing this entry
  struct cds_list_head queueNode;  ///< fdQ node
  DataEnc_MetaInfoBuffer(15) meta; ///< MetaInfo (FinalBlockId only)
  uint64_t version;                ///< version number
  uint64_t lastSeg;                ///< last segment number
  int fd;                          ///< file descriptor
  uint16_t refcnt;                 ///< number of inflight SQEs referencing this entry
  uint16_t prefixL;                ///< mount+path TLV-LENGTH
  uint16_t versionedL;             ///< mount+path+[32=ls]+version TLV-LENGTH
  uint16_t segmentL;               ///< mount+path+[32=ls]+version+finalSeg TLV-LENGTH
  uint8_t nameV[NameMaxLength];    ///< name TLV-VALUE
} FileServerFd;

/** @brief Sentinel value to indicate file not found. */
extern FileServerFd* FileServer_NotFound;

typedef struct FileServer FileServer;

/**
 * @brief Open or reference file descriptor.
 * @param name Interest name.
 * @retval NULL filename is invalid or does not match a mount.
 * @retval FileServer_NotFound file does not exist; do not Unref this value.
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
  /** @brief Required bits in @c stx_mask . */
  FileServerStatxRequired = STATX_TYPE | STATX_MODE | STATX_MTIME | STATX_SIZE,
  /** @brief Optional bits in @c stx_mask . */
  FileServerStatxOptional = STATX_ATIME | STATX_CTIME | STATX_BTIME,
};

/** @brief Determine whether @c stx_mask has requested bits. */
static __rte_always_inline bool
FileServerFd_HasStatBit(const FileServerFd* entry, uint32_t bit)
{
  return (entry->st.stx_mask & bit) == bit;
}

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

/** @brief Convert @c statx_timestamp to nanoseconds. */
static __rte_always_inline uint64_t
FileServerFd_StatTime(struct statx_timestamp t)
{
  return (uint64_t)t.tv_sec * SPDK_SEC_TO_NSEC + (uint64_t)t.tv_nsec;
}

/**
 * @brief Encode metadata packet payload.
 * @param entry a valid FileServerFd entry.
 * @param payload payload mbuf.
 */
__attribute__((nonnull)) bool
FileServerFd_EncodeMetadata(FileServer* p, FileServerFd* entry, struct rte_mbuf* payload);

/**
 * @brief Encode directory listing.
 * @param entry a valid FileServerFd entry.
 * @param payload payload mbuf.
 * @param segmentLen maximum payload length.
 * @pre FileServerFd_IsDir(entry) is true.
 *
 * This is a limited implementation that returns one segment a subset of directory entries that
 * can fit in one segment. It also depends on dirent.d_type field being available.
 */
__attribute__((nonnull)) bool
FileServerFd_EncodeLs(FileServer* p, FileServerFd* entry, struct rte_mbuf* payload,
                      uint16_t segmentLen);

#endif // NDNDPDK_FILESERVER_FD_H
