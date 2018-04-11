#include "fwd.h"
#include "token.h"

#include "../../container/pcct/pit-dn-up-it.h"
#include "../../core/logger.h"

INIT_ZF_LOG(FwFwd);

static bool
FwFwd_VerifyNack(FwFwd* fwd, Packet* npkt, PitEntry* pitEntry, PitUp** up,
                 int* nPending, NackReason* leastSevereReason)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  PNack* nack = Packet_GetNackHdr(npkt);

  if (unlikely(pitEntry == NULL)) {
    ZF_LOGD("^ drop=no-PIT-entry");
    return false;
  }

  *up = NULL;
  *nPending = 0;
  *leastSevereReason = nack->lpl3.nackReason;

  PitUpIt it;
  for (PitUpIt_Init(&it, pitEntry); PitUpIt_Valid(&it); PitUpIt_Next(&it)) {
    if (it.up->face == FACEID_INVALID) {
      break;
    }
    if (it.up->face == pkt->port) {
      *up = it.up;
      continue;
    }
    if (it.up->nack == NackReason_None) {
      ++(*nPending);
    } else {
      *leastSevereReason = NackReason_GetMin(*leastSevereReason, it.up->nack);
    }
  }
  if (unlikely(*up == NULL)) {
    return false;
  }

  if (unlikely((*up)->nonce != nack->interest.nonce)) {
    ZF_LOGD("^ drop=wrong-nonce pit-nonce=%" PRIx32 " up-nonce=%" PRIx32,
            (*up)->nonce, nack->interest.nonce);
    return false;
  }

  return true;
}

static bool
FwFwd_RxNackCongestion(FwFwd* fwd, PitEntry* pitEntry, PitUp* up,
                       TscTime rxTime)
{
  TscTime now = rte_get_tsc_cycles();

  uint32_t upNonce = up->nonce;
  PitUp_AddRejectedNonce(up, upNonce);
  bool hasAltNonce = PitUp_ChooseNonce(up, pitEntry, now, &upNonce);
  if (!hasAltNonce) {
    return false;
  }

  uint32_t upLifetime = PitEntry_GetTxInterestLifetime(pitEntry, now);
  uint8_t hopLimit = 0xFF; // TODO properly set HopLimit
  Packet* outNpkt =
    ModifyInterest(pitEntry->npkt, upNonce, upLifetime, hopLimit, fwd->headerMp,
                   fwd->guiderMp, fwd->indirectMp);
  if (unlikely(outNpkt == NULL)) {
    ZF_LOGD("^ no-interest-to=%" PRI_FaceId " drop=alloc-error", up->face);
    return true;
  }

  uint64_t token = FwToken_New(fwd->id, Pit_GetEntryToken(fwd->pit, pitEntry));
  Packet_InitLpL3Hdr(outNpkt)->pitToken = token;
  Packet_ToMbuf(outNpkt)->timestamp = rxTime; // for latency stats

  ZF_LOGD("^ interest-to=%" PRI_FaceId " npkt=%p nonce-%08" PRIu32
          " up-token=%016" PRIx64,
          up->face, outNpkt, upNonce, token);
  Face_Tx(up->face, outNpkt);
  return true;
}

void
FwFwd_RxNack(FwFwd* fwd, Packet* npkt)
{
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  uint64_t token = Packet_GetLpL3Hdr(npkt)->pitToken;
  PNack* nack = Packet_GetNackHdr(npkt);
  TscTime rxTime = pkt->timestamp;
  NackReason reason = nack->lpl3.nackReason;

  ZF_LOGD("nack-from=%" PRI_FaceId " npkt=%p up-token=%016" PRIx64
          " reason=%" PRIu8,
          pkt->port, npkt, token, reason);

  // find PIT entry and verify nonce in Nack matches nonce in PitUp
  PitEntry* pitEntry = Pit_FindByNack(fwd->pit, npkt);
  PitUp* up;
  int nPending;
  NackReason leastSevereReason;
  bool ok =
    FwFwd_VerifyNack(fwd, npkt, pitEntry, &up, &nPending, &leastSevereReason);
  rte_pktmbuf_free(pkt);
  npkt = NULL;
  pkt = NULL;
  nack = NULL;
  if (unlikely(!ok)) {
    return;
  }

  // record NackReason in PitUp
  up->nack = reason;

  // for Duplicate, resend with an alternate nonce if available
  if (reason == NackReason_Duplicate &&
      FwFwd_RxNackCongestion(fwd, pitEntry, up, rxTime)) {
    return;
  }

  // if other upstream are pending, wait for them
  if (nPending > 0) {
    ZF_LOGD("^ drop=more-pending(%d)", nPending);
    return;
  }

  // return Nacks to downstream
  PitDnIt it;
  for (PitDnIt_Init(&it, pitEntry); PitDnIt_Valid(&it); PitDnIt_Next(&it)) {
    PitDn* dn = it.dn;
    if (dn->face == FACEID_INVALID) {
      break;
    }
    if (dn->expiry < rxTime) {
      continue;
    }

    if (unlikely(Face_IsDown(dn->face))) {
      ZF_LOGD("^ no-data-to=%" PRI_FaceId " drop=face-down", dn->face);
      continue;
    }

    Packet* outNpkt =
      ModifyInterest(pitEntry->npkt, dn->nonce, 0, 0, fwd->headerMp,
                     fwd->guiderMp, fwd->indirectMp);
    if (unlikely(outNpkt == NULL)) {
      ZF_LOGD("^ no-nack-to=%" PRI_FaceId " drop=alloc-error", up->face);
      break;
    }
    MakeNack(outNpkt, leastSevereReason);
    Packet_GetLpL3Hdr(outNpkt)->pitToken = dn->token;
    Face_Tx(dn->face, outNpkt);
  }

  // erase PIT entry
  Pit_Erase(fwd->pit, pitEntry);
}
