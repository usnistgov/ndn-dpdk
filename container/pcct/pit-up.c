#include "pit-up.h"
#include "pit-dn-up-it.h"

static bool
PitUp_HasRejectedNonce(PitUp* up, uint32_t nonce)
{
  for (int i = 0; i < PIT_UP_MAX_REJ_NONCES; ++i) {
    if (up->rejectedNonces[i] == nonce) {
      return true;
    }
  }
  return false;
}

bool
PitUp_ChooseNonce(PitUp* up, PitEntry* entry, TscTime now, uint32_t* nonce)
{
  if (likely(!PitUp_HasRejectedNonce(up, *nonce))) {
    return true;
  }

  PitDnIt it;
  for (PitDnIt_Init(&it, entry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
    if (dn->face == FACEID_INVALID) {
      break;
    }
    if (dn->expiry < now) {
      continue;
    }
    if (!PitUp_HasRejectedNonce(up, dn->nonce)) {
      *nonce = dn->nonce;
      return true;
    }
  }
  return false;
}
