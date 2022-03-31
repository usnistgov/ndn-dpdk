#include "cs-disk.h"
#include "../disk/alloc.h"
#include "../disk/store.h"
#include "cs-arc.h"

#include "../core/logger.h"

N_LOG_INIT(CsDisk);

void
CsDisk_Insert(Cs* cs, CsEntry* entry)
{
  uint64_t slot = DiskAlloc_Alloc(cs->diskAlloc);
  if (unlikely(slot == 0)) {
    N_LOGD("Insert entry=%p data=%p" N_LOG_ERROR("no-slot"), entry, entry->data);
    CsEntry_Clear(entry);
    ++cs->nDiskFull;
    return;
  }

  NDNDPDK_ASSERT(entry->kind == CsEntryMemory);
  bool ok = DiskStore_PutPrepare(cs->diskStore, entry->data, &entry->diskStored);
  if (unlikely(!ok)) {
    N_LOGW("Insert error entry=%p data=%p" N_LOG_ERROR("prepare"), entry, entry->data);
    CsEntry_Clear(entry);
    return;
  }

  N_LOGD("Insert entry=%p data=%p slot=%" PRIu64 " sp-pktLen=%" PRIu16 " sp-saveTotal=%" PRIu16,
         entry, entry->data, slot, entry->diskStored.pktLen, entry->diskStored.saveTotal);
  DiskStore_PutData(cs->diskStore, slot, entry->data, &entry->diskStored);
  entry->kind = CsEntryDisk;
  entry->diskSlot = slot;
  ++cs->nDiskInsert;
}

void
CsDisk_Delete(Cs* cs, CsEntry* entry)
{
  N_LOGD("Delete entry=%p slot=%" PRIu64, entry, entry->diskSlot);
  NDNDPDK_ASSERT(entry->kind == CsEntryDisk);
  DiskAlloc_Free(cs->diskAlloc, entry->diskSlot);
  entry->kind = CsEntryNone;
  ++cs->nDiskDelete;
}

void
CsDisk_ArcMove(CsEntry* entry, CsListID src, CsListID dst, uintptr_t ctx)
{
  Cs* cs = (Cs*)ctx;
  switch (CsArc_MoveDir(src, dst)) {
    case CsArc_MoveDirC(T1, B1):
      CsEntry_Clear(entry);
      break;
    case CsArc_MoveDirC(T2, B2):
      CsDisk_Insert(cs, entry);
      break;
    case CsArc_MoveDirC(B2, T2):
    case CsArc_MoveDirC(B2, Del):
      if (entry->kind == CsEntryDisk) {
        CsDisk_Delete(cs, entry);
      }
      break;
    case CsArc_MoveDirC(New, T1):
    case CsArc_MoveDirC(T1, T2):
    case CsArc_MoveDirC(B1, T2):
    case CsArc_MoveDirC(B1, Del):
    case CsArc_MoveDirC(T1, Del):
      break;
    default:
      NDNDPDK_ASSERT(false);
  }
}
