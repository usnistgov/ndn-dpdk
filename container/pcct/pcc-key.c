#include "pcc-key.h"
#include "debug-string.h"

const char*
PccSearch_ToDebugString(const PccSearch* search)
{
  PccDebugString_Clear();

  char nameStr[LNAME_MAX_STRING_SIZE + 1];
  if (LName_ToString(search->name, nameStr, sizeof(nameStr)) == 0) {
    snprintf(nameStr, sizeof(nameStr), "(empty)");
  }
  PccDebugString_Appendf("name=%s", nameStr);

  if (LName_ToString(search->fh, nameStr, sizeof(nameStr)) == 0) {
    snprintf(nameStr, sizeof(nameStr), "(empty)");
  }
  return PccDebugString_Appendf(" fh=%s", nameStr);
}

void
PccKey_CopyFromSearch(PccKey* key, const PccSearch* search)
{
  assert(search->name.length <= sizeof(key->name));
  assert(search->fh.length <= sizeof(key->fh));
  rte_memcpy(key->name, search->name.value, search->name.length);
  rte_memcpy(key->fh, search->fh.value, search->fh.length);
}
