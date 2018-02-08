#include "pcc-key.h"

void
PccKey_CopyFromSearch(PccKey* key, const PccSearch* search)
{
  assert(search->name.length <= sizeof(key->name));
  assert(search->fh.length <= sizeof(key->fh));
  rte_memcpy(key->name, search->name.value, search->name.length);
  rte_memcpy(key->fh, search->fh.value, search->fh.length);
}
