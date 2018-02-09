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
        unsigned numaSocket, uint8_t startDepth)
{
  Fib* fib = Tsht_New(id, maxEntries, nBuckets, Fib_LookupMatch,
                      sizeof(FibEntry), sizeof(FibPriv), numaSocket);

  FibPriv* fibp = Fib_GetPriv(fib);
  fibp->startDepth = startDepth;

  return fib;
}

FibEntry*
Fib_Alloc(Fib* fib)
{
  FibEntry* entry = Tsht_AllocT(fib, FibEntry);
  if (likely(entry != NULL)) {
    memset(entry, 0, sizeof(*entry));
  }
  return entry;
}

bool
Fib_Insert(Fib* fib, FibEntry* entry)
{
  LName key;
  key.length = entry->nameL;
  key.value = entry->nameV;
  uint64_t hash = LName_ComputeHash(key);

  return Tsht_Insert(fib, hash, &key, entry);
}

const FibEntry*
Fib_Find(Fib* fib, uint16_t nameL, const uint8_t* nameV)
{
  LName key;
  key.length = nameL;
  key.value = nameV;
  uint64_t hash = LName_ComputeHash(key);

  return Tsht_FindT(fib, hash, &key, FibEntry);
}

static const FibEntry*
Fib_GetEntryByPrefix(Fib* fib, const Name* name, LName lname,
                     uint16_t prefixLen)
{
  lname.length = Name_GetPrefixSize(name, prefixLen);
  uint64_t hash = Name_ComputePrefixHash(name, prefixLen);
  return Tsht_FindT(fib, hash, &lname, FibEntry);
}

const FibEntry*
Fib_Lpm(Fib* fib, const Name* name)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  uint8_t scratch[NAME_MAX_LENGTH];
  LName lname = Name_Linearize(name, scratch);

  const FibEntry* startEntry = NULL;
  int prefixLen = name->nComps;
  if (fibp->startDepth < prefixLen) {
    startEntry = Fib_GetEntryByPrefix(fib, name, lname, fibp->startDepth);
    if (startEntry == NULL) {
      prefixLen = fibp->startDepth - 1;
    } else if (startEntry->maxDepth > 0) {
      prefixLen += startEntry->maxDepth;
      if (prefixLen > name->nComps) {
        prefixLen = name->nComps;
      }
    }
  }

  for (; prefixLen >= 0; --prefixLen) {
    const FibEntry* entry =
      unlikely(prefixLen == fibp->startDepth)
        ? startEntry
        : Fib_GetEntryByPrefix(fib, name, lname, prefixLen);
    if (entry != NULL && entry->nNexthops > 0) {
      return entry;
    }
  }

  return NULL;
}
