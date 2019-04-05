#ifndef NDN_DPDK_CONTAINER_PCCT_PIT_DN_UP_IT_H
#define NDN_DPDK_CONTAINER_PCCT_PIT_DN_UP_IT_H

/// \file

#include "pit-entry.h"

typedef struct __PitDnUpIt
{
  union
  {
    PitDn* dn; ///< current PitDn
    PitUp* up; ///< current PitUp
  };
  int index; ///< index of PitDn/PitUp

  int i;   ///< (pvt) index within this array
  int max; ///< (pvt) upper bound of this array
  union
  {
    void* array; // (pvt) start of array
    PitDn* dns;
    PitUp* ups;
  };

  PitEntryExt** nextPtr; ///< (pvt) next extension
} __PitDnUpIt;

static void
__PitDnUpIt_Init(__PitDnUpIt* it,
                 PitEntry* entry,
                 int maxInEntry,
                 size_t offsetInEntry)
{
  it->index = 0;
  it->i = 0;
  it->max = maxInEntry;
  it->array = RTE_PTR_ADD(entry, offsetInEntry);
  it->nextPtr = &entry->ext;
}

static void
__PitDnUpIt_Next(__PitDnUpIt* it, int maxInExt, size_t offsetInExt)
{
  assert(it->i < it->max);
  ++it->index;
  ++it->i;
  if (likely(it->i < it->max)) {
    return;
  }

  PitEntryExt* ext = *it->nextPtr;
  if (ext == NULL) {
    return;
  }
  it->i = 0;
  it->max = maxInExt;
  it->array = RTE_PTR_ADD(ext, offsetInExt);
  it->nextPtr = &ext->next;
}

bool
__PitDnUpIt_Extend(__PitDnUpIt* it, Pit* pit, int maxInExt, size_t offsetInExt);

/** \brief Iterator of DN slots in PIT entry.
 *
 *  \code
 *  PitDnIt it;
 *  for (PitDnIt_Init(&it, entry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
 *    int index = it.index;
 *    PitDn* dn = it.dn;
 *  }
 *  \endcode
 */
typedef __PitDnUpIt PitDnIt;

static void
PitDnIt_Init(PitDnIt* it, PitEntry* entry)
{
  __PitDnUpIt_Init(it, entry, PIT_ENTRY_MAX_DNS, offsetof(PitEntry, dns));
  it->dn = &it->dns[it->i];
}

static bool
PitDnIt_Valid(PitDnIt* it)
{
  return it->i < it->max;
}

static void
PitDnIt_Next(PitDnIt* it)
{
  __PitDnUpIt_Next(it, PIT_ENTRY_EXT_MAX_DNS, offsetof(PitEntryExt, dns));
  it->dn = &it->dns[it->i];
}

/** \brief Add an extension for more DN slots.
 *  \retval true extension added, iterator points to next slot.
 *  \retval false unable to allocate extension
 */
static bool
PitDnIt_Extend(PitDnIt* it, Pit* pit)
{
  bool ok = __PitDnUpIt_Extend(
    it, pit, PIT_ENTRY_EXT_MAX_DNS, offsetof(PitEntryExt, dns));
  it->dn = &it->dns[it->i];
  return ok;
}

/** \brief Iterator of UP slots in PIT entry.
 *
 *  \code
 *  PitUpIt it;
 *  for (PitUpIt_Init(&it, entry); PitUpIt_Valid(&it); PitUpIt_Next(&it)) {
 *    int index = it.index;
 *    PitUp* up = it.up;
 *  }
 *  \endcode
 */
typedef __PitDnUpIt PitUpIt;

static void
PitUpIt_Init(PitUpIt* it, PitEntry* entry)
{
  __PitDnUpIt_Init(it, entry, PIT_ENTRY_MAX_UPS, offsetof(PitEntry, ups));
  it->up = &it->ups[it->i];
}

static bool
PitUpIt_Valid(PitUpIt* it)
{
  return it->i < it->max;
}

static void
PitUpIt_Next(PitUpIt* it)
{
  __PitDnUpIt_Next(it, PIT_ENTRY_EXT_MAX_UPS, offsetof(PitEntryExt, ups));
  it->up = &it->ups[it->i];
}

/** \brief Add an extension for more UP slots.
 *  \retval true extension added, iterator points to next slot.
 *  \retval false unable to allocate extension
 */
static bool
PitUpIt_Extend(PitDnIt* it, Pit* pit)
{
  bool ok = __PitDnUpIt_Extend(
    it, pit, PIT_ENTRY_EXT_MAX_UPS, offsetof(PitEntryExt, ups));
  it->up = &it->ups[it->i];
  return ok;
}

#endif // NDN_DPDK_CONTAINER_PCCT_PIT_DN_UP_IT_H
