#include "pit-entry.h"
#include "../core/base16.h"
#include "../core/logger.h"
#include "pit-iterator.h"
#include "pit.h"

N_LOG_INIT(PitEntry);

static_assert(sizeof(PitEntryExt) <= sizeof(PccEntry), "");

enum {
  PitDebugStringLength = Base16_BufferSize(NameMaxLength) +
                         6 * (PitMaxDns + PitMaxExtDns + PitMaxUps + PitMaxExtUps) + 32,
};
static RTE_DEFINE_PER_LCORE(
  struct { char buffer[PitDebugStringLength]; }, PitDebugStringBuffer);

const char*
PitEntry_ToDebugString(PitEntry* entry) {
  int pos = 0;
#define buffer (RTE_PER_LCORE(PitDebugStringBuffer).buffer)
#define append(fn, ...)                                                                            \
  do {                                                                                             \
    pos += fn(RTE_PTR_ADD(buffer, pos), PitDebugStringLength - pos, __VA_ARGS__);                  \
  } while (false)

  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  append(Base16_Encode, interest->name.value, interest->name.length);

  if (entry->nCanBePrefix > 0) {
    append(snprintf, "[P%" PRIu8 "]", entry->nCanBePrefix);
  }
  if (entry->mustBeFresh) {
    append(snprintf, "[F]");
  }

  append(snprintf, ",DN[");
  {
    const char* delim = "";
    PitDn_Each (it, entry, false) {
      if (it.dn->face == 0) {
        break;
      }
      if (it.index >= PitMaxDns + PitMaxExtDns) {
        append(snprintf, "%s...", delim);
        break;
      }
      append(snprintf, "%s%" PRI_FaceID, delim, it.dn->face);
      delim = " ";
    }
  }

  append(snprintf, "],UP[");
  {
    const char* delim = "";
    PitUp_Each (it, entry, false) {
      if (it.up->face == 0) {
        break;
      }
      if (it.index >= PitMaxUps + PitMaxExtUps) {
        append(snprintf, "%s...", delim);
        break;
      }
      append(snprintf, "%s%" PRI_FaceID, delim, it.up->face);
      delim = " ";
    }
  }
  append(snprintf, "]");

  return buffer;
#undef buffer
#undef append
}

FibEntry*
PitEntry_FindFibEntry(PitEntry* entry, Fib* fib) {
  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  LName name = {.length = entry->fibPrefixL, .value = interest->name.value};
  if (unlikely(interest->activeFwHint >= 0)) {
    name.value = interest->fwHint.value;
  }
  FibEntry* fibEntry = Fib_Find(fib, name, entry->fibPrefixHash);
  if (unlikely(fibEntry == NULL || fibEntry->seqNum != entry->fibSeqNum)) {
    return NULL;
  }
  return fibEntry;
}

void
PitEntry_SetExpiryTimer(PitEntry* entry, Pit* pit) {
  entry->hasSgTimer = false;
  bool ok = MinTmr_At(&entry->timeout, entry->expiry, pit->timeoutSched);
  NDNDPDK_ASSERT(ok); // unless PIT_MAX_LIFETIME is higher than scheduler limit
}

bool
PitEntry_SetSgTimer(PitEntry* entry, Pit* pit, TscDuration after) {
  if (rte_get_tsc_cycles() + after > entry->expiry) {
    return false;
  }
  entry->hasSgTimer = true;
  bool ok = MinTmr_After(&entry->timeout, after, pit->timeoutSched);
  if (unlikely(!ok)) {
    PitEntry_SetExpiryTimer(entry, pit);
  }
  return ok;
}

void
PitEntry_Timeout_(MinTmr* tmr, uintptr_t pitPtr) {
  Pit* pit = (Pit*)pitPtr;
  PitEntry* entry = container_of(tmr, PitEntry, timeout);
  if (entry->hasSgTimer) {
    N_LOGD("Timeout(sgtimer) pit=%p pit-entry=%p", pit, entry);
    PitEntry_SetExpiryTimer(entry, pit);
    pit->sgTimerCb(pit, entry, pit->sgTimerCtx);
  } else {
    N_LOGD("Timeout(expiry) pit=%p pit-entry=%p", pit, entry);
    ++pit->nExpired;
    Pit_Erase(pit, entry);
  }
}

FaceID
PitEntry_FindDuplicateNonce(PitEntry* entry, uint32_t nonce, FaceID dnFace) {
  PitDn_Each (it, entry, false) {
    PitDn* dn = it.dn;
    if (dn->face == 0) {
      break;
    }
    if (dn->face == dnFace) {
      continue;
    }
    if (dn->nonce == nonce) {
      return dn->face;
    }
  }
  return 0;
}

__attribute__((nonnull)) static inline PitDn*
PitEntry_ReserveDn(PitEntry* entry, FaceID face, TscTime now) {
  PitDn* dn = &entry->dns[0];
  if (likely(dn->face == 0)) { // first DN record in new PIT entry
    entry->dns[1].face = 0;
    goto NEW;
  }

  PitDn_Each (it, entry, true) {
    dn = it.dn;
    if (dn->face == face) {
      return dn;
    }
    if (dn->face == 0) {
      PitDn_UseSlot(&it);
      goto NEW;
    }
    if (dn->expiry < now) {
      NDNDPDK_ASSERT(entry->nCanBePrefix >= (uint8_t)dn->canBePrefix);
      entry->nCanBePrefix -= (uint8_t)dn->canBePrefix;
      goto NEW;
    }
  }
  return NULL;
NEW:
  POISON(dn);
  dn->face = face;
  return dn;
}

PitDn*
PitEntry_InsertDn(PitEntry* entry, Pit* pit, Packet* npkt) {
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceID face = pkt->port;
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  PitDn* dn = PitEntry_ReserveDn(entry, face, Mbuf_GetTimestamp(pkt));
  if (unlikely(dn == NULL)) {
    return NULL;
  }

  // refresh DN record
  dn->token = lpl3->pitToken;
  dn->congMark = lpl3->congMark;
  dn->canBePrefix = interest->canBePrefix;
  dn->nonce = interest->nonce;
  uint32_t lifetime = RTE_MIN(interest->lifetime, PIT_MAX_LIFETIME);
  dn->expiry = Mbuf_GetTimestamp(pkt) + TscDuration_FromMillis(lifetime);

  // update txHopLimit
  NDNDPDK_ASSERT(interest->hopLimit > 0); // decoder rejects HopLimit=0
  entry->txHopLimit = RTE_MAX(entry->txHopLimit, interest->hopLimit - 1);

  // record CanBePrefix and prefer CBP=1 for representative Interest
  if (entry->npkt == npkt) {
    // first DN record, entry->npkt and entry->nCanBePrefix are assigned in PitEntry_Init
  } else {
    entry->nCanBePrefix += (uint8_t)interest->canBePrefix;
    if (entry->nCanBePrefix == (uint8_t)interest->canBePrefix) {
      // this happens in two situations:
      // A. two conditions, both are met:
      //    1. interest->canBePrefix is true
      //    2. entry->nCanBePrefix was 0 before this Interest
      // B. three conditions, all are met:
      //    1. interest->canBePrefix is false
      //    2. entry->nCanBePrefix was 1 before this Interest
      //    3. entry->nCanBePrefix is decremented in PitEntry_ReserveDn due to expired DN record
      rte_pktmbuf_free(Packet_ToMbuf(entry->npkt));
      entry->npkt = npkt;
    } else {
      rte_pktmbuf_free(pkt);
    }
  }
  NULLize(npkt);
  NULLize(pkt);
  NULLize(interest);

  // set expiry timer
  if (dn->expiry > entry->expiry) {
    entry->expiry = dn->expiry;
    PitEntry_SetExpiryTimer(entry, pit);
  }

  return dn;
}

PitUp*
PitEntry_FindUp(PitEntry* entry, FaceID face) {
  PitUp_Each (it, entry, false) {
    PitUp* up = it.up;
    if (up->face == face) {
      return up;
    }
    if (up->face == 0) {
      break;
    }
  }
  return NULL;
}

PitUp*
PitEntry_ReserveUp(PitEntry* entry, FaceID face) {
  PitUp* up = NULL;
  PitUp_Each (it, entry, true) {
    up = it.up;
    if (up->face == face) {
      return up;
    }
    if (up->face == 0) {
      PitUp_UseSlot(&it);
      goto NEW;
    }
    if (up->lastTx == 0) {
      goto NEW;
    }
  }
  return NULL;
NEW:
  PitUp_Reset(up, face);
  return up;
}
