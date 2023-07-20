#ifndef NDNDPDK_CORE_MMAPFD_H
#define NDNDPDK_CORE_MMAPFD_H

/** @file */

#include "common.h"

/** @brief Memory map and file descriptor. */
typedef struct MmapFd {
  void* map;
  size_t size;
  int fd;
} MmapFd;

/**
 * @brief Create a file with memory map.
 * @param size intended file size.
 * @return whether success.
 */
__attribute__((nonnull)) bool
MmapFd_Open(MmapFd* m, const char* filename, size_t size);

/**
 * @brief Close a file with memory map.
 * @param size truncate to final file size.
 * @return whether success.
 */
__attribute__((nonnull)) bool
MmapFd_Close(MmapFd* m, const char* filename, size_t size);

/**
 * @brief Access mapped memory region.
 * @pre @c MmapFd_Open was successful.
 */
__attribute__((nonnull, returns_nonnull)) __rte_always_inline void*
MmapFd_At(const MmapFd* m, size_t pos) {
  return RTE_PTR_ADD(m->map, pos);
}

#endif // NDNDPDK_CORE_MMAPFD_H
