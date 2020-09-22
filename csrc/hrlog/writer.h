#ifndef NDNDPDK_HRLOG_WRITER_H
#define NDNDPDK_HRLOG_WRITER_H

/** @file */

#include "../dpdk/thread.h"
#include "entry.h"

typedef struct HrlogWriter
{
  ;
} HrlogWriter;

/**
 * @brief Write high resolution logs to a file.
 * @param nSkip how many initial entries to discard.
 * @param nTotal how many entries to collect.
 */
int
Hrlog_RunWriter(const char* filename, int nSkip, int nTotal, ThreadStopFlag* stop);

#endif // NDNDPDK_HRLOG_WRITER_H
