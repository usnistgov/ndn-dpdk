#ifndef NDNDPDK_PDUMP_WRITER_H
#define NDNDPDK_PDUMP_WRITER_H

/** @file */

#include "../core/mmapfd.h"
#include "../dpdk/thread.h"

/** @brief Packet dump writer. */
typedef struct PdumpWriter
{
  ThreadCtrl ctrl;
  struct rte_ring* queue;
  const char* filename;
  MmapFd m;
  size_t maxSize;
  size_t pos;
  uint32_t nextIntf;
  uint32_t intf[UINT16_MAX + 1];
} PdumpWriter;

__attribute__((nonnull)) int
PdumpWriter_Run(PdumpWriter* w);

#endif // NDNDPDK_PDUMP_WRITER_H
