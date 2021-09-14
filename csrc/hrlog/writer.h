#ifndef NDNDPDK_HRLOG_WRITER_H
#define NDNDPDK_HRLOG_WRITER_H

/** @file */

#include "../dpdk/thread.h"
#include "entry.h"

/** @brief High resolution log writer task. */
typedef struct HrlogWriter
{
  ThreadCtrl ctrl;
  const char* filename;
  int64_t nSkip;  ///< how many initial entries to discard
  int64_t nTotal; ///< how many entries to collect
} HrlogWriter;

/** @brief Write high resolution logs to a file. */
bool
Hrlog_RunWriter(HrlogWriter* w);

#endif // NDNDPDK_HRLOG_WRITER_H
