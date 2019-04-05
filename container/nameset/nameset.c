#include "nameset.h"

typedef struct NameSetRecord
{
  uint16_t nameL;
  uint8_t nameV[0];
} NameSetRecord;

static void*
NameSetRecord_GetUsr(NameSetRecord* record)
{
  return RTE_PTR_ADD(record->nameV, record->nameL);
}

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
__NameSet_Insert(NameSet* set,
                 uint16_t nameL,
                 const uint8_t* nameV,
                 const void* usr,
                 size_t usrLen)
{
  NameSetRecord* record =
    rte_zmalloc_socket("NameSetRecord",
                       sizeof(NameSetRecord) + nameL + usrLen,
                       0,
                       set->numaSocket);
  assert(record != NULL);
  record->nameL = nameL;
  rte_memcpy(record->nameV, nameV, nameL);
  if (usrLen > 0 && usr != NULL) {
    rte_memcpy(NameSetRecord_GetUsr(record), usr, usrLen);
  }

  ++set->nRecords;

  if (set->records == NULL) {
    set->records = rte_malloc_socket("NameSetRecords",
                                     set->nRecords * sizeof(NameSetRecord*),
                                     0,
                                     set->numaSocket);
  } else {
    set->records =
      rte_realloc(set->records, set->nRecords * sizeof(NameSetRecord*), 0);
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

LName
NameSet_GetName(const NameSet* set, int index)
{
  assert(index >= 0 && index < set->nRecords);
  NameSetRecord* record = set->records[index];
  LName name = { .length = record->nameL, .value = record->nameV };
  return name;
}

void*
NameSet_GetUsr(const NameSet* set, int index)
{
  assert(index >= 0 && index < set->nRecords);
  NameSetRecord* record = set->records[index];
  return NameSetRecord_GetUsr(record);
}

int
__NameSet_FindExact(const NameSet* set, uint16_t nameL, const uint8_t* nameV)
{
  for (int i = 0; i < set->nRecords; ++i) {
    NameSetRecord* record = set->records[i];
    if (record->nameL == nameL &&
        memcmp(record->nameV, nameV, record->nameL) == 0) {
      return i;
    }
  }
  return -1;
}

int
__NameSet_FindPrefix(const NameSet* set, uint16_t nameL, const uint8_t* nameV)
{
  for (int i = 0; i < set->nRecords; ++i) {
    NameSetRecord* record = set->records[i];
    if (record->nameL <= nameL &&
        memcmp(record->nameV, nameV, record->nameL) == 0) {
      return i;
    }
  }
  return -1;
}
