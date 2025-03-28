#include "rxtable.h"
#include "../core/logger.h"
#include "face.h"

N_LOG_INIT(EthRxTable);

void
EthRxTable_Init(EthRxTable* rxt, uint16_t port) {
  rxt->base.rxBurst = EthRxTable_RxBurst;
  rxt->base.rxThread = 0;
  rxt->port = port;
  rxt->queue = 0;
  CDS_INIT_LIST_HEAD(&rxt->head);
}

__attribute__((nonnull)) static inline bool
EthRxTable_Accept(EthRxTable* rxt, struct rte_mbuf* m) {
  EthRxMatchResult match = 0;
  EthFacePriv* priv = NULL;
  const char* acceptMethod = NULL;

#define CHECK_GTP_TUNNEL_MATCH(execMatch)                                                          \
  (priv->rxMatch.act == EthRxMatchActGtp) && __extension__({                                       \
    if (execMatch) {                                                                               \
      match = EthRxMatch_MatchGtpInner(&priv->rxMatch, m);                                         \
    }                                                                                              \
    !(match & EthRxMatchResultHit) && (match & EthRxMatchResultGtp);                               \
  })

  FaceID id = (FaceID)Mbuf_GetMark(m);
  if (id != 0) {
    Face* face = Face_Get(id);
    if (likely(face->impl != NULL)) {
      priv = Face_GetPriv(face);
      acceptMethod = "mark";
      if (CHECK_GTP_TUNNEL_MATCH(true)) {
        // rte_flow only matches GTPv1 header but does not check inner IPv4+UDP headers.
        // If the inner headers mismatch, the packet belongs to the pass-through face.
        goto GTP;
      }
      goto ACCEPT;
    }
  }

  const EthXdpHdr* xh = rte_pktmbuf_mtod(m, const EthXdpHdr*);
  if (likely(m->data_len >= sizeof(*xh)) && xh->magic == UINT64_MAX) {
    Face* face = Face_Get((FaceID)(xh->fmv >> 16));
    if (likely(face->impl != NULL) &&
        likely((priv = Face_GetPriv(face))->rxMatch.len == xh->hdrLen)) {
      acceptMethod = "xdphdr";
      goto ACCEPT;
    }
  }

  acceptMethod = "iter";
  // RCU lock is inherited from RxLoop_Run
  struct cds_list_head* pos;
  cds_list_for_each_rcu (pos, &rxt->head) {
    priv = container_of(pos, EthFacePriv, rxtNode);
    match = EthRxMatch_Match(&priv->rxMatch, m);
    if (match & EthRxMatchResultHit) {
      goto ACCEPT;
    }
    if (CHECK_GTP_TUNNEL_MATCH(false)) {
      // For a GTP-U face, if outer headers up to GTPv1 match but inner IPv4+UDP headers mismatch,
      // the packet is dispatched to the pass-through face for potential GTP-IP uplink processing.
      // As an optimization, the matched FaceID is saved in mbuf mark, so that GTP-IP handler can
      // skip table lookup.
      Mbuf_SetMark(m, priv->faceID);
      goto GTP;
    }
  }
  return false;

ACCEPT:
  N_LOGD("accepting to face rxt=%p mbuf=%p method=%s face=%" PRI_FaceID, rxt, m, acceptMethod,
         priv->faceID);
  m->port = priv->faceID;
  rte_pktmbuf_adj(m, priv->rxMatch.len);
  return true;

GTP:
  // RCU lock is inherited from RxLoop_Run
  pos = rcu_dereference(rxt->head.prev);
  if (unlikely(pos == NULL)) {
    return false;
  }
  EthFacePriv* passthruPriv = container_of(pos, EthFacePriv, rxtNode);
  if (passthruPriv->rxMatch.act != EthRxMatchActAlways || passthruPriv->rxMatch.len != 0) {
    // not a passthru face
    return false;
  }
  N_LOGD("accepting to passthru rxt=%p mbuf=%p method=%s gtp-face=%" PRI_FaceID
         " passthru=%" PRI_FaceID,
         rxt, m, acceptMethod, priv->faceID, passthruPriv->faceID);
  m->port = passthruPriv->faceID;
  return true;

#undef CHECK_GTP_TUNNEL_MATCH
}

void
EthRxTable_RxBurst(RxGroup* rxg, RxGroupBurstCtx* ctx) {
  EthRxTable* rxt = container_of(rxg, EthRxTable, base);
  ctx->nRx = rte_eth_rx_burst(rxt->port, rxt->queue, ctx->pkts, RTE_DIM(ctx->pkts));
  uint64_t now = rte_get_tsc_cycles();

  PdumpEthPortUnmatchedCtx unmatch;
  // RCU lock is inherited from RxLoop_Run
  PdumpEthPortUnmatchedCtx_Init(&unmatch, rxt->port);

  struct rte_mbuf* bounceBufs[MaxBurstSize];
  uint16_t nBounceBufs = 0;
  for (uint16_t i = 0; i < ctx->nRx; ++i) {
    struct rte_mbuf* m = ctx->pkts[i];
    Mbuf_SetTimestamp(m, now);
    if (unlikely(!EthRxTable_Accept(rxt, m))) {
      RxGroupBurstCtx_Drop(ctx, i);
      if (PdumpEthPortUnmatchedCtx_Append(&unmatch, m)) {
        ctx->pkts[i] = NULL;
      } else if (rxt->copyTo != NULL) {
        // free bounce bufs locally instead of via RxLoop, because rte_pktmbuf_free_bulk is most
        // efficient when consecutive mbufs are from the same mempool such as the main mempool
        bounceBufs[nBounceBufs++] = m;
        ctx->pkts[i] = NULL;
      }
      continue;
    }

    if (rxt->copyTo == NULL) {
      continue;
    }

    ctx->pkts[i] = rte_pktmbuf_copy(m, rxt->copyTo, 0, UINT32_MAX);
    if (unlikely(ctx->pkts[i] == NULL)) {
      RxGroupBurstCtx_Drop(ctx, i);
    }
    bounceBufs[nBounceBufs++] = m;
  }

  PdumpEthPortUnmatchedCtx_Process(&unmatch);
  if (unlikely(nBounceBufs > 0)) {
    rte_pktmbuf_free_bulk(bounceBufs, nBounceBufs);
  }
}

STATIC_ASSERT_FUNC_TYPE(RxGroup_RxBurstFunc, EthRxTable_RxBurst);
