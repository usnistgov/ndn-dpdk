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
  EthFacePriv* priv = NULL;

  FaceID id = EthFace_RxMbufFaceID(m);
  if (id != 0) {
    Face* face = Face_Get(id);
    if (likely(face->impl != NULL)) {
      priv = Face_GetPriv(face);

      if (priv->rxMatch.act == EthRxMatchActGtp) {
        EthRxMatchResult match = EthRxMatch_Match(&priv->rxMatch, m);
        if (!(match & EthRxMatchResultHit) && (match & EthRxMatchResultGtp)) {
          struct cds_list_head* passthruPos = rcu_dereference(rxt->head.prev);
          EthFacePriv* passthruPriv = NULL;
          if (passthruPos != NULL &&
              (passthruPriv = container_of(passthruPos, EthFacePriv, rxtNode))->rxMatch.len == 0) {
            priv = passthruPriv;
          }
        }
      }

      goto ACCEPT;
    }
  }

  const EthXdpHdr* xh = rte_pktmbuf_mtod(m, const EthXdpHdr*);
  if (likely(m->data_len >= sizeof(*xh)) && xh->magic == UINT64_MAX) {
    Face* face = Face_Get((FaceID)(xh->fmv >> 16));
    if (likely(face->impl != NULL) &&
        likely((priv = Face_GetPriv(face))->rxMatch.len == xh->hdrLen)) {
      goto ACCEPT;
    }
  }

  // RCU lock is inherited from RxLoop_Run
  struct cds_list_head* pos;
  cds_list_for_each_rcu (pos, &rxt->head) {
    priv = container_of(pos, EthFacePriv, rxtNode);
    EthRxMatchResult match = EthRxMatch_Match(&priv->rxMatch, m);
    if (match & EthRxMatchResultHit) {
      goto ACCEPT;
    }
  }
  return false;

ACCEPT:
  m->port = priv->faceID;
  rte_pktmbuf_adj(m, priv->rxMatch.len);
  return true;
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
