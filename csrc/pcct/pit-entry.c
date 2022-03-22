#include "pit-entry.h"
#include "pit-iterator.h"
#include "pit.h"

#include "../core/logger.h"

N_LOG_INIT(PitEntry);

static_assert(sizeof(PitEntryExt) <= sizeof(PccEntry), "");

enum
{
  PitDebugStringLength =
    NameHexBufferLength + 6 * (PitMaxDns + PitMaxExtDns + PitMaxUps + PitMaxExtUps) + 32,
};
static RTE_DEFINE_PER_LCORE(
  struct { char buffer[PitDebugStringLength]; }, PitDebugStringBuffer);

const char*
PitEntry_ToDebugString(PitEntry* entry)
{
  int pos = 0;
#define buffer (RTE_PER_LCORE(PitDebugStringBuffer).buffer)
#define append(...)                                                                                \
  do {                                                                                             \
    pos += snprintf(RTE_PTR_ADD(buffer, pos), PitDebugStringLength - pos, __VA_ARGS__);            \
  } while (false)

  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  pos += LName_PrintHex(PName_ToLName(&interest->name), RTE_PTR_ADD(buffer, pos));

  if (entry->nCanBePrefix > 0) {
    append("[P%" PRIu8 "]", entry->nCanBePrefix);
  }
  if (entry->mustBeFresh) {
    append("[F]");
  }

  append(",DN[");
  {
    const char* delim = "";
    PitDnIt it;
    for (PitDnIt_Init(&it, entry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
      if (it.dn->face == 0) {
        break;
      }
      if (it.index >= PitMaxDns + PitMaxExtDns) {
        append("%s...", delim);
        break;
      }
      append("%s%" PRI_FaceID, delim, it.dn->face);
      delim = " ";
    }
  }

  append("],UP[");
  {
    const char* delim = "";
    PitUpIt it;
    for (PitUpIt_Init(&it, entry); PitUpIt_Valid(&it); PitUpIt_Next(&it)) {
      if (it.up->face == 0) {
        break;
      }
      if (it.index >= PitMaxUps + PitMaxExtUps) {
        append("%s...", delim);
        break;
      }
      append("%s%" PRI_FaceID, delim, it.up->face);
      delim = " ";
    }
  }
  append("]");

  return buffer;
#undef buffer
#undef append
}

FibEntry*
PitEntry_FindFibEntry(PitEntry* entry, Fib* fib)
{
  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  LName name = { .length = entry->fibPrefixL, .value = interest->name.value };
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
PitEntry_SetExpiryTimer(PitEntry* entry, Pit* pit)
{
  entry->hasSgTimer = false;
  bool ok = MinTmr_At(&entry->timeout, entry->expiry, pit->timeoutSched);
  NDNDPDK_ASSERT(ok); // unless PIT_MAX_LIFETIME is higher than scheduler limit
}

bool
PitEntry_SetSgTimer(PitEntry* entry, Pit* pit, TscDuration after)
{
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
PitEntry_Timeout_(MinTmr* tmr, uintptr_t pitPtr)
{
  Pit* pit = (Pit*)pitPtr;
  PitEntry* entry = container_of(tmr, PitEntry, timeout);
  if (entry->hasSgTimer) {
    N_LOGD("Timeout(sgtimer) pit=%p pit-entry=%p", pit, entry);
    PitEntry_SetExpiryTimer(entry, pit);
    pit->sgTimerCb(pit, entry, pit->sgTimerCtx);
  } else {
    N_LOGD("Timeout(expiry) pit=%p pit-entry=%p", pit, entry);
    Pit_Erase(pit, entry);
  }
}

FaceID
PitEntry_FindDuplicateNonce(PitEntry* entry, uint32_t nonce, FaceID dnFace)
{
  PitDnIt it;
  for (PitDnIt_Init(&it, entry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
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

PitDn*
PitEntry_InsertDn(PitEntry* entry, Pit* pit, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceID face = pkt->port;
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  PitDn* dn = NULL;
  if (entry->npkt == npkt) { // new PIT entry
    dn = &entry->dns[0];
    NDNDPDK_ASSERT(dn->face == 0);
    dn->face = face;
  } else { // find DN slot
    PitDnIt it;
    for (PitDnIt_Init(&it, entry); PitDnIt_Valid(&it) || PitDnIt_Extend(&it, pit);
         PitDnIt_Next(&it)) {
      dn = it.dn;
      if (dn->face == face) {
        break;
      }
      if (dn->face == 0) {
        dn->face = face;
        break;
      }
      if (dn->expiry < Mbuf_GetTimestamp(pkt)) {
        NDNDPDK_ASSERT(entry->nCanBePrefix >= (uint8_t)dn->canBePrefix);
        entry->nCanBePrefix -= (uint8_t)dn->canBePrefix;
        dn->face = face;
        break;
      }
    }
    if (unlikely(!PitDnIt_Valid(&it))) {
      return NULL;
    }
  }

  // refresh DN record
  dn->token = lpl3->pitToken;
  dn->congMark = lpl3->congMark;
  dn->canBePrefix = interest->canBePrefix;
  dn->nonce = interest->nonce;
  uint32_t lifetime = RTE_MIN(interest->lifetime, PIT_MAX_LIFETIME);
  dn->expiry = Mbuf_GetTimestamp(pkt) + TscDuration_FromMillis(lifetime);

  // record CanBePrefix and prefer CBP=1 for representative Interest
  if (entry->nCanBePrefix != (uint8_t)interest->canBePrefix) {
    NDNDPDK_ASSERT(entry->npkt != npkt);
    rte_pktmbuf_free(Packet_ToMbuf(entry->npkt));
    entry->npkt = npkt;
  } else if (entry->npkt != npkt) {
    rte_pktmbuf_free(pkt);
  }
  entry->nCanBePrefix += (uint8_t)interest->canBePrefix;

  // update txHopLimit
  NDNDPDK_ASSERT(interest->hopLimit > 0); // decoder rejects HopLimit=0
  entry->txHopLimit = RTE_MAX(entry->txHopLimit, interest->hopLimit - 1);

  // set expiry timer
  if (dn->expiry > entry->expiry) {
    entry->expiry = dn->expiry;
    PitEntry_SetExpiryTimer(entry, pit);
  }

  return dn;
}

PitUp*
PitEntry_FindUp(PitEntry* entry, FaceID face)
{
  PitUpIt it;
  for (PitUpIt_Init(&it, entry); PitUpIt_Valid(&it); PitUpIt_Next(&it)) {
    PitUp* up = it.up;
    if (up->face == face) {
      return up;
    }
  }
  return NULL;
}

PitUp*
PitEntry_ReserveUp(PitEntry* entry, Pit* pit, FaceID face)
{
  PitUpIt it;
  for (PitUpIt_Init(&it, entry); PitUpIt_Valid(&it) || PitUpIt_Extend(&it, pit);
       PitUpIt_Next(&it)) {
    PitUp* up = it.up;
    if (up->face == face) {
      return up;
    }
    if (up->face == 0 || up->lastTx == 0) {
      PitUp_Reset(up, face);
      return up;
    }
  }
  return NULL;
}
