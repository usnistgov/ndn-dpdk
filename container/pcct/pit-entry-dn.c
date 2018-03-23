#include "pit-entry.h"
#include "pit.h"

int
PitEntry_DnRxInterest(Pit* pit, PitEntry* entry, Packet* npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceId face = pkt->port;
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  int index;
  PitDn* dn = NULL;
  if (entry->npkt == npkt) { // new PIT entry
    index = 0;
    dn = &entry->dns[0];
    assert(dn->face == FACEID_INVALID);
    dn->face = face;
  } else { // find DN slot
    for (index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
      dn = &entry->dns[index];
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
    if (unlikely(dn == NULL)) {
      return -1;
    }
  }

  // refresh downstream record
  dn->token = lpl3->pitToken;
  dn->canBePrefix = interest->canBePrefix;
  dn->nonce = interest->nonce;
  uint32_t lifetime = interest->lifetime <= PIT_MAX_LIFETIME
                        ? interest->lifetime
                        : PIT_MAX_LIFETIME;
  dn->expiry = pkt->timestamp + lifetime * rte_get_tsc_hz() / 1000;

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

  entry->lastDnIndex = index;
  return index;
}
