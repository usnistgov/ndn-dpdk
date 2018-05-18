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

static void
Fib_FinalizeEntry(TshtEntryPtr entry0, Tsht* tsht)
{
  FibEntry* entry = (FibEntry*)entry0;
  Fib* fib = (Fib*)tsht;
  if (likely(entry->dyn != NULL) && entry->shouldFreeDyn) {
    struct rte_mempool* dynMp = rte_mempool_from_obj(entry->dyn);
    rte_mempool_put(dynMp, entry->dyn);
  }
  if (likely(entry->strategy != NULL)) {
    StrategyCode_Unref(entry->strategy);
  }
}

Fib*
Fib_New(const char* id, uint32_t maxEntries, uint32_t nBuckets,
        unsigned numaSocket, uint8_t startDepth)
{
  Fib* fib =
    (Fib*)Tsht_New(id, maxEntries, nBuckets, Fib_LookupMatch, Fib_FinalizeEntry,
                   sizeof(FibEntry), sizeof(FibPriv), numaSocket);

  FibPriv* fibp = Fib_GetPriv(fib);
  fibp->startDepth = startDepth;

  return fib;
}

bool
Fib_Insert(Fib* fib, FibEntry* entry)
{
  if (likely(entry->strategy != NULL)) {
    StrategyCode_Ref(entry->strategy);
    assert(entry->dyn != NULL);
  } else {
    assert(entry->nNexthops == 0);
  }

  LName key;
  key.length = entry->nameL;
  key.value = entry->nameV;
  uint64_t hash = LName_ComputeHash(key);

  return Tsht_Insert(Fib_ToTsht(fib), hash, &key, entry);
}

static const FibEntry*
Fib_GetEntryByPrefix(Fib* fib, const PName* name, const uint8_t* nameV,
                     uint16_t prefixLen)
{
  uint64_t hash = PName_ComputePrefixHash(name, nameV, prefixLen);
  LName key = {.length = PName_SizeofPrefix(name, nameV, prefixLen),
               .value = nameV };
  return Tsht_FindT(Fib_ToTsht(fib), hash, &key, FibEntry);
}

const FibEntry*
__Fib_Lpm(Fib* fib, const PName* name, const uint8_t* nameV)
{
  FibPriv* fibp = Fib_GetPriv(fib);

  // first stage
  int prefixLen = name->nComps;
  if (fibp->startDepth < prefixLen) {
    const FibEntry* entry =
      Fib_GetEntryByPrefix(fib, name, nameV, fibp->startDepth);
    if (entry == NULL) { // continue to shorter prefixes
      prefixLen = fibp->startDepth - 1;
    } else if (entry->maxDepth > 0) { // restart at a longest prefix
      prefixLen = fibp->startDepth + entry->maxDepth;
      if (prefixLen > name->nComps) {
        prefixLen = name->nComps;
      }
    } else if (entry->nNexthops > 0) { // the start entry itself is a match
      return entry;
    }
  }

  // second stage
  for (; prefixLen >= 0; --prefixLen) {
    const FibEntry* entry = Fib_GetEntryByPrefix(fib, name, nameV, prefixLen);
    if (entry != NULL && entry->nNexthops > 0) {
      return entry;
    }
  }

  return NULL;
}
