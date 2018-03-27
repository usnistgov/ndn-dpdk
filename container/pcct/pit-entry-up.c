#include "pit-entry.h"
#include "pit.h"

int
PitEntry_UpTxInterest(Pit* pit, PitEntry* entry, FaceId face, Packet** npkt)
{
  PitPriv* pitp = Pit_GetPriv(pit);
  *npkt = NULL;

  // find UP slot
  int index;
  PitUp* up = NULL;
  for (index = 0; index < PIT_ENTRY_MAX_UPS; ++index) {
    up = &entry->ups[index];
    if (up->face == face) {
      break;
    }
    if (up->face == FACEID_INVALID) {
      PitUp_Reset(up, face);
      break;
    }
  }
  if (unlikely(up == NULL)) {
    return -1;
  }

  // prepare UP record
  up->lastTx = rte_get_tsc_cycles();
  up->canBePrefix = (bool)entry->nCanBePrefix;
  up->nack = NackReason_None;

  // choose nonce
  up->nonce = entry->dns[entry->lastDnIndex].nonce;
  if (unlikely(PitUp_HasRejectedNonce(up, up->nonce))) {
    for (int dnIndex = 0; dnIndex < PIT_ENTRY_MAX_DNS; ++dnIndex) {
      PitDn* dn = &entry->dns[dnIndex];
      if (dn->face == FACEID_INVALID) {
        break;
      }
      if (!PitUp_HasRejectedNonce(up, dn->nonce)) {
        up->nonce = dn->nonce;
        break;
      }
    }
  }

  // prepare outgoing Interest
  uint32_t lifetime = TscDuration_ToMillis(entry->expiry - up->lastTx);
  *npkt = ModifyInterest(entry->npkt, up->nonce, lifetime, pitp->headerMp,
                         pitp->guiderMp, pitp->indirectMp);

  return index;
}
