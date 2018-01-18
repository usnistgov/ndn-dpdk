#include "facetable.h"

int
FaceTable_Count(FaceTable* ft)
{
  return atomic_load_explicit(&ft->count, memory_order_relaxed);
}

Face*
FaceTable_GetFace(FaceTable* ft, FaceId id)
{
  return atomic_load_explicit(&ft->table[id], memory_order_relaxed);
}

void
FaceTable_SetFace(FaceTable* ft, Face* face)
{
  assert(face->id != FACEID_INVALID);
  Face* oldFace =
    atomic_exchange_explicit(&ft->table[face->id], face, memory_order_relaxed);
  assert(oldFace == NULL);
  atomic_fetch_add_explicit(&ft->count, 1, memory_order_relaxed);
}

void
FaceTable_UnsetFace(FaceTable* ft, FaceId id)
{
  Face* oldFace =
    atomic_exchange_explicit(&ft->table[id], NULL, memory_order_relaxed);
  if (oldFace != NULL) {
    atomic_fetch_sub_explicit(&ft->count, 1, memory_order_relaxed);
  }
}
