#include "pit-entry.h"
#include "pit.h"

static void
PitEntry_ScheduleTimeout(Pit* pit, PitEntry* entry)
{
  PitPriv* pitp = Pit_GetPriv(pit);

  // determine PIT entry expiry time: the last expiry time among PitDns
  TscTime expiry = 0;
  for (int index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
    PitDn* dn = &entry->dns[index];
    if (dn->face == FACEID_INVALID) {
      break;
    }
    if (dn->expiry > expiry) {
      expiry = dn->expiry;
    }
  }
  assert(expiry > 0);

  MinTmr_Cancel(&entry->timeout);
  bool ok = MinTmr_At(&entry->timeout, expiry, pitp->timeoutSched);
  assert(ok);
}

int
PitEntry_DnRxInterest(Pit* pit, PitEntry* entry, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  FaceId face = pkt->port;
  LpL3* lpl3 = Packet_GetLpL3Hdr(npkt);
  PInterest* interest = Packet_GetInterestHdr(npkt);

  // find slot for downstream record
  int index;
  PitDn* dn = NULL;
  if (entry->npkt == NULL) {
    entry->npkt = npkt;
    index = 0;
    dn = &entry->dns[0];
    assert(dn->face == FACEID_INVALID);
    dn->face = face;
  } else {
    for (index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
      dn = &entry->dns[index];
      if (dn->face == face) {
        break;
      }
      if (dn->face == FACEID_INVALID) {
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

  // put CanBePrefix on outgoing Interests if any downstream specifies that
  if (!entry->canBePrefix && interest->canBePrefix) {
    assert(entry->npkt != npkt);
    rte_pktmbuf_free(Packet_ToMbuf(entry->npkt));
    entry->npkt = npkt;
    entry->canBePrefix = true;
  } else {
    rte_pktmbuf_free(Packet_ToMbuf(npkt));
  }

  PitEntry_ScheduleTimeout(pit, entry);
}
