#ifndef NDNDPDK_PCCT_CS_DISK_H
#define NDNDPDK_PCCT_CS_DISK_H

/** @file */

#include "cs-struct.h"

__attribute__((nonnull)) void
CsDisk_Insert(Cs* cs, CsEntry* entry);

__attribute__((nonnull)) void
CsDisk_Delete(Cs* cs, CsEntry* entry);

__attribute__((nonnull)) void
CsDisk_ArcMove(CsEntry* entry, CsListID src, CsListID dst, uintptr_t ctx);

#endif // NDNDPDK_PCCT_CS_DISK_H
