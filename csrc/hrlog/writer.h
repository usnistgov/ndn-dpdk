#ifndef NDNDPDK_HRLOG_WRITER_H
#define NDNDPDK_HRLOG_WRITER_H

/** @file */

#include "../dpdk/thread.h"
#include "entry.h"

/** @brief High resolution log writer task. */
typedef struct HrlogWriter
{
  ThreadCtrl ctrl;
  struct rte_ring* queue;
  const char* filename;
  int64_t count;
} HrlogWriter;

/** @brief Write high resolution logs to a file. */
int
HrlogWriter_Run(HrlogWriter* w);

#endif // NDNDPDK_HRLOG_WRITER_H
