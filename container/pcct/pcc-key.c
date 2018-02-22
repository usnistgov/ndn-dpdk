#include "pcc-key.h"

typedef struct PccSearchDebugString
{
  char s[5 + LNAME_MAX_STRING_SIZE + 4 + LNAME_MAX_STRING_SIZE + 1];
} PccSearchDebugString;
RTE_DEFINE_PER_LCORE(PccSearchDebugString, gPccSearchDebugString);

const char*
PccSearch_ToDebugString(const PccSearch* search)
{
  char nameStr[LNAME_MAX_STRING_SIZE + 1];
  if (LName_ToString(search->name, nameStr, sizeof(nameStr)) == 0) {
    snprintf(nameStr, sizeof(nameStr), "(empty)");
  }
  char fhStr[LNAME_MAX_STRING_SIZE + 1];
  if (LName_ToString(search->fh, nameStr, sizeof(nameStr)) == 0) {
    snprintf(fhStr, sizeof(fhStr), "(empty)");
  }

  PccSearchDebugString* ds = &RTE_PER_LCORE(gPccSearchDebugString);
  snprintf(ds->s, sizeof(ds->s), "name=%s fh=%s", nameStr, fhStr);
  return ds->s;
}

void
PccKey_CopyFromSearch(PccKey* key, const PccSearch* search)
{
  assert(search->name.length <= sizeof(key->name));
  assert(search->fh.length <= sizeof(key->fh));
  rte_memcpy(key->name, search->name.value, search->name.length);
  rte_memcpy(key->fh, search->fh.value, search->fh.length);
}
