#include "pit-entry.h"
#include "debug-string.h"
#include "pit-dn-up-it.h"
#include "pit.h"

static_assert(sizeof(PitEntryExt) <= sizeof(PccEntry), "");

const char*
PitEntry_ToDebugString(PitEntry* entry)
{
  PccDebugString_Clear();

  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  char nameStr[LNAME_MAX_STRING_SIZE + 1];
  if (LName_ToString(*(LName*)&interest->name, nameStr, sizeof(nameStr)) == 0) {
    snprintf(nameStr, sizeof(nameStr), "(empty)");
  }

  PccDebugString_Appendf("%s CBP=%" PRIu8 " MBF=%d DN=[", nameStr,
                         entry->nCanBePrefix, (int)entry->mustBeFresh);
  for (int index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
    PitDn* dn = &entry->dns[index];
    if (dn->face == FACEID_INVALID) {
      break;
    }
    PccDebugString_Appendf("%" PRI_FaceId ",", dn->face);
  }
  PccDebugString_Appendf("] UP=[");
  for (int index = 0; index < PIT_ENTRY_MAX_UPS; ++index) {
    PitUp* up = &entry->ups[index];
    if (up->face == FACEID_INVALID) {
      break;
    }
    PccDebugString_Appendf("%" PRI_FaceId ",", up->face);
  }
  return PccDebugString_Appendf("]");
}

PitDn*
PitEntry_InsertDn(PitEntry* entry, Pit* pit, Packet* npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceId face = pkt->port;
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  PitDn* dn = NULL;
  if (entry->npkt == npkt) { // new PIT entry
    dn = &entry->dns[0];
    assert(dn->face == FACEID_INVALID);
    dn->face = face;
  } else { // find DN slot
    PitDnIt it;
    for (PitDnIt_Init(&it, entry);
         PitDnIt_Valid(&it) || PitDnIt_Extend(&it, pit); PitDnIt_Next(&it)) {
      dn = it.dn;
      if (dn->face == face) {
        break;
      }
      if (dn->face == FACEID_INVALID) {
        dn->face = face;
        break;
      }
      if (dn->expiry < pkt->timestamp) {
        assert(entry->nCanBePrefix >= (uint8_t)dn->canBePrefix);
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
  dn->canBePrefix = interest->canBePrefix;
  dn->nonce = interest->nonce;
  uint32_t lifetime = interest->lifetime <= PIT_MAX_LIFETIME
                        ? interest->lifetime
                        : PIT_MAX_LIFETIME;
  dn->expiry = pkt->timestamp + TscDuration_FromMillis(lifetime);

  // record CanBePrefix and prefer CBP=1 for representative Interest
  if (entry->nCanBePrefix != (uint8_t)interest->canBePrefix) {
    assert(entry->npkt != npkt);
    rte_pktmbuf_free(Packet_ToMbuf(entry->npkt));
    entry->npkt = npkt;
  } else if (entry->npkt != npkt) {
    rte_pktmbuf_free(pkt);
  }
  entry->nCanBePrefix += (uint8_t)interest->canBePrefix;

  // set timer
  if (dn->expiry > entry->expiry) {
    entry->expiry = dn->expiry;
    MinTmr_Cancel(&entry->timeout);
    bool ok = MinTmr_At(&entry->timeout, entry->expiry, pitp->timeoutSched);
    assert(ok); // unless PIT_MAX_LIFETIME is higher than scheduler limit
  }

  return dn;
}

PitUp*
PitEntry_ReserveUp(PitEntry* entry, Pit* pit, FaceId face)
{
  PitUpIt it;
  for (PitUpIt_Init(&it, entry); PitUpIt_Valid(&it) || PitUpIt_Extend(&it, pit);
       PitUpIt_Next(&it)) {
    PitUp* up = it.up;
    if (up->face == face) {
      return up;
    }
    if (up->face == FACEID_INVALID || up->lastTx == 0) {
      PitUp_Reset(up, face);
      return up;
    }
  }
  return NULL;
}
