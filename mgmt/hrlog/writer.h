#ifndef NDN_DPDK_MGMT_HRLOG_WRITER_H
#define NDN_DPDK_MGMT_HRLOG_WRITER_H

/// \file

#include "post.h"

/** \brief Write high resolution logs to a file.
 *  \param nSkip how many initial entries to discard.
 *  \param nTotal how many entries to collect.
 */
int
Hrlog_RunWriter(const char* filename, int nSkip, int nTotal);

#endif // NDN_DPDK_MGMT_HRLOG_WRITER_H
