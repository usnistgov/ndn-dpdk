#include "fib.h"

#include "../../ndn/namehash.h"

static bool
Fib_LookupMatch(TshtMatchNode node, const void* key0)
{
  const FibEntry* entry = TshtMatch_GetEntry(node, FibEntry);
  const LName* key = (const LName*)key0;
  return entry->nameL == key->length &&
         memcmp(entry->nameV, key->value, key->length) == 0;
}

Fib*
Fib_New(const char* id, uint32_t maxEntries, uint32_t nBuckets,
        unsigned numaSocket)
{
  Fib* fib = Tsht_New(id, maxEntries, nBuckets, Fib_LookupMatch,
                      sizeof(FibEntry), sizeof(FibPriv), numaSocket);
  return fib;
}

void
Fib_Close(Fib* fib)
{
  assert(false); // not implemented
}

FibInsertResult
Fib_Insert(Fib* fib, const FibEntry* entry)
{
  FibEntry* newEntry = Tsht_AllocT(fib, FibEntry);
  if (newEntry == NULL) {
    return FIB_INSERT_ALLOC_ERROR;
  }
  rte_memcpy(newEntry, entry, sizeof(*newEntry));

  LName key;
  key.length = newEntry->nameL;
  key.value = newEntry->nameV;
  uint64_t hash = LName_ComputeHash(key);

  rcu_read_lock();
  bool res = Tsht_Insert(fib, hash, &key, newEntry);
  rcu_read_unlock();
  return res;
}

bool
Fib_Erase(Fib* fib, uint16_t nameL, const uint8_t* nameV)
{
  LName key;
  key.length = nameL;
  key.value = nameV;
  uint64_t hash = LName_ComputeHash(key);

  bool ok = false;
  rcu_read_lock();
  FibEntry* entry = Tsht_FindT(fib, hash, &key, FibEntry);
  if (entry != NULL) {
    ok = Tsht_Erase(fib, entry);
  }
  rcu_read_unlock();
  return ok;
}

const FibEntry*
Fib_Lpm(Fib* fib, const Name* name)
{
  assert(false); // not implemented
  return NULL;
}
