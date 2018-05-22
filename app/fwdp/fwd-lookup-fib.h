#ifndef NDN_DPDK_APP_FWDP_FWD_LOOKUP_FIB_H
#define NDN_DPDK_APP_FWDP_FWD_LOOKUP_FIB_H

/// \file

#include "fwd.h"

static const FibEntry*
FwFwd_LookupFibByInterest(FwFwd* fwd, Packet* npkt, FibNexthopFilter* nhFlt)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  FaceId dnFace = Packet_ToMbuf(npkt)->port;

  if (likely(interest->nFhs == 0)) {
    const FibEntry* entry = Fib_Lpm(fwd->fib, &interest->name);
    if (unlikely(entry == NULL)) {
      return NULL;
    }
    *nhFlt = 0;
    int nNexthops = FibNexthopFilter_Reject(nhFlt, entry, dnFace);
    if (unlikely(nNexthops == 0)) {
      return NULL;
    }
    return entry;
  }

  for (int fhIndex = 0; fhIndex < interest->nFhs; ++fhIndex) {
    NdnError e = PInterest_SelectActiveFh(interest, fhIndex);
    if (unlikely(e != NdnError_OK)) {
      // caller would treat this as "no FIB match" and reply Nack
      return false;
    }

    const FibEntry* entry = Fib_Lpm(fwd->fib, &interest->activeFhName);
    if (unlikely(entry == NULL)) {
      continue;
    }
    *nhFlt = 0;
    int nNexthops = FibNexthopFilter_Reject(nhFlt, entry, dnFace);
    if (unlikely(nNexthops == 0)) {
      continue;
    }
    return entry;
  }
  return NULL;
}

static const FibEntry*
FwFwd_LookupFibByPitEntry(FwFwd* fwd, PitEntry* pitEntry)
{
  PccEntry* pccEntry = PccEntry_FromPitEntry(pitEntry);
  Name name;
  uint32_t nameL;
  if (unlikely(pccEntry->key.fhL != 0)) {
    name.v = pccEntry->key.fhV;
    nameL = pccEntry->key.fhL;
  } else {
    name.v = pccEntry->key.nameV;
    nameL = pccEntry->key.nameL;
  }

  // TODO avoid reparsing and hash computation
  NdnError res = PName_Parse(&name.p, nameL, name.v);
  assert(res == NdnError_OK);

  return Fib_Lpm(fwd->fib, &name);
}

#endif // NDN_DPDK_APP_FWDP_FWD_LOOKUP_FIB_H
