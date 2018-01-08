#include "nameset.h"
#include <rte_malloc.h>

typedef struct NameSetRecord
{
  uint32_t len;
} NameSetRecord;

#define NameSetRecord_GetComps(record) (((uint8_t*)(record)) + sizeof(uint32_t))

void
NameSet_Close(NameSet* set)
{
  for (int i = 0; i < set->nRecords; ++i) {
    rte_free(set->records[i]);
  }

  if (set->records != NULL) {
    rte_free(set->records);
  }
}

void
NameSet_Insert(NameSet* set, const uint8_t* comps, uint32_t compsLen)
{
  NameSetRecord* record =
    rte_malloc("NameSetRecord", sizeof(uint32_t) + compsLen, 0);
  assert(record != NULL);
  record->len = compsLen;
  rte_memcpy(NameSetRecord_GetComps(record), comps, compsLen);

  ++set->nRecords;

  if (set->records == NULL) {
    set->records =
      rte_malloc("NameSetRecords", set->nRecords * sizeof(set->records[0]), 0);
  } else {
    set->records =
      rte_realloc(set->records, set->nRecords * sizeof(set->records[0]), 0);
  }
  assert(set->records != NULL);

  set->records[set->nRecords - 1] = record;
}

void
NameSet_Erase(NameSet* set, int index)
{
  assert(index >= 0 && index < set->nRecords);
  rte_free(set->records[index]);
  set->records[index] = set->records[--set->nRecords];
}

const uint8_t*
NameSet_GetName(const NameSet* set, int index, uint32_t* compsLen)
{
  assert(index >= 0 && index < set->nRecords);
  NameSetRecord* record = set->records[index];
  *compsLen = record->len;
  return NameSetRecord_GetComps(record);
}

int
NameSet_FindExact(const NameSet* set, const uint8_t* comps, uint32_t compsLen)
{
  for (int i = 0; i < set->nRecords; ++i) {
    NameSetRecord* record = set->records[i];
    if (record->len == compsLen &&
        memcmp(NameSetRecord_GetComps(record), comps, record->len) == 0) {
      return i;
    }
  }
  return -1;
}

int
NameSet_FindPrefix(const NameSet* set, const uint8_t* comps, uint32_t compsLen)
{
  for (int i = 0; i < set->nRecords; ++i) {
    NameSetRecord* record = set->records[i];
    if (record->len <= compsLen &&
        memcmp(NameSetRecord_GetComps(record), comps, record->len) == 0) {
      return i;
    }
  }
  return -1;
}
