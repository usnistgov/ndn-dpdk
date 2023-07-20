#include "server.h"

#include "../core/logger.h"
#include "../ndni/tlv-encoder.h"
#include "fd.h"
#include "naming.h"
#include "op.h"

N_LOG_INIT(FileServer);

static uint8_t NameComponent_Segment0[3];
static uint8_t MetaInfo_Metadata[16];
static uint8_t MetaInfo_Nack[16];

RTE_INIT(InitMetaInfo) {
  NameComponent_Segment0[0] = TtSegmentNameComponent;
  NameComponent_Segment0[1] = 1;
  NameComponent_Segment0[2] = 0;
  DataEnc_PrepareMetaInfo(MetaInfo_Metadata, ContentBlob, FileServerMetadataFreshness,
                          (LName){.length = 3, .value = NameComponent_Segment0});
  DataEnc_PrepareMetaInfo(MetaInfo_Nack, ContentNack, FileServerMetadataFreshness, (LName){0});
}

typedef struct RxBurstCtx {
  TscTime now;
  uint16_t index;     ///< interest[:index] are processed
  uint16_t nInterest; ///< interest[index:nInterest] are unprocessed
  uint16_t nData;     ///< data[:nData] are to be transmitted
  struct rte_mbuf* interest[MaxBurstSize];
  Packet* data[MaxBurstSize];
} RxBurstCtx;

__attribute__((nonnull)) static __rte_always_inline bool
FileServerRx_CheckVersion(FileServer* p, FileServerFd* fd, FileServerRequestName rn) {
  if (likely(rn.version == fd->version)) {
    return true;
  }
  return p->versionBypassHi != 0 && (rn.version >> 32) == p->versionBypassHi;
}

__attribute__((nonnull)) static void
FileServerRx_Read(FileServer* p, RxBurstCtx* ctx, FileServerRequestName rn) {
  ++p->cnt.reqRead;
  Packet* interest = Packet_FromMbuf(ctx->interest[ctx->index]);
  PInterest* pi = Packet_GetInterestHdr(interest);

  FileServerFd* fd = FileServerFd_Open(p, &pi->name, ctx->now);
  if (unlikely(fd == NULL)) {
    N_LOGD("Read drop=no-fd");
    return;
  }
  if (unlikely(fd == FileServer_NotFound)) {
    N_LOGD("Read drop=file-not-found");
    return;
  }
  if (unlikely(!FileServerFd_IsFile(fd))) {
    N_LOGD("Read fd=%d drop=mode-not-file", fd->fd);
    goto UNREF;
  }
  if (unlikely(!FileServerRx_CheckVersion(p, fd, rn))) {
    N_LOGD("Read fd=%d drop=version-changed rn-version=%" PRIu64 " fd-version=%" PRIu64, fd->fd,
           rn.version, fd->version);
    goto UNREF;
  }
  if (unlikely(rn.segment > fd->lastSeg)) {
    N_LOGD("Read fd=%d drop=segment-out-of-range rn-segment=%" PRIu64 " lastseg=%" PRIu64, fd->fd,
           rn.segment, fd->lastSeg);
    goto UNREF;
  }

  FileServerOp* op = NULL;
  if (unlikely(rte_mempool_get(p->opMp, (void**)&op) != 0)) {
    N_LOGW("Read fd=%d drop=no-op", fd->fd);
    goto UNREF;
  }
  op->fd = fd;
  op->segment = rn.segment;
  uint64_t contentOffset = rn.segment * p->segmentLen;
  op->contentLen = RTE_MIN((uint64_t)p->segmentLen, fd->st.stx_size - contentOffset);
  op->data = DataEnc_EncodeRoom(PName_ToLName(&pi->name), (LName){0}, fd->meta, op->contentLen,
                                op->iov, &op->iovcnt, &p->mp, Face_PacketTxAlign(p->face));
  if (unlikely(op->data == NULL)) {
    N_LOGW("Read fd=%d drop=no-data", fd->fd);
    goto FREE_OP;
  }

  struct io_uring_sqe* sqe = Uring_GetSqe(&p->ur);
  if (unlikely(sqe == NULL)) {
    N_LOGW("Read fd=%d drop=no-sqe" N_LOG_ERROR_BLANK, fd->fd);
    ctx->index = ctx->nInterest; // drop subsequent Interests too
    goto FREE_DATA;
  }
  N_LOGV("Read fd=%d segment=%" PRIu64 " iovcnt=%d", fd->fd, op->segment, op->iovcnt);
  io_uring_prep_readv(sqe, fd->fd, op->iov, op->iovcnt, contentOffset);
  io_uring_sqe_set_data(sqe, op);
  op->interestL3 = *Packet_GetLpL3Hdr(interest);
  return;

FREE_DATA:
  rte_pktmbuf_free(op->data);
FREE_OP:
  rte_mempool_put(p->opMp, op);
UNREF:
  FileServerFd_Unref(p, fd);
  NULLize(fd);
}

__attribute__((nonnull)) static void
FileServerRx_Ls(FileServer* p, RxBurstCtx* ctx, FileServerRequestName rn) {
  ++p->cnt.reqLs;
  Packet* interest = Packet_FromMbuf(ctx->interest[ctx->index]);
  PInterest* pi = Packet_GetInterestHdr(interest);

  FileServerFd* fd = FileServerFd_Open(p, &pi->name, ctx->now);
  if (unlikely(fd == NULL)) {
    N_LOGD("Ls drop=no-fd");
    return;
  }
  if (unlikely(fd == FileServer_NotFound)) {
    N_LOGD("Ls drop=not-found");
    return;
  }
  if (unlikely(!FileServerFd_IsDir(fd))) {
    N_LOGD("Ls fd=%d drop=mode-not-dir", fd->fd);
    goto UNREF;
  }
  if (unlikely(!FileServerRx_CheckVersion(p, fd, rn))) {
    N_LOGD("Ls fd=%d drop=version-changed rn-version=%" PRIu64 " fd-version=%" PRIu64, fd->fd,
           rn.version, fd->version);
    goto UNREF;
  }
  if (fd->lsL == UINT32_MAX) {
    bool ok = FileServerFd_GenerateLs(p, fd);
    if (unlikely(!ok)) {
      N_LOGD("Ls fd=%d drop=ls-error", fd->fd);
      goto UNREF;
    }
  }
  if (unlikely(rn.segment > fd->lastSeg)) {
    N_LOGD("Ls fd=%d drop=segment-out-of-range rn-segment=%" PRIu64 " lastseg=%" PRIu64, fd->fd,
           rn.segment, fd->lastSeg);
    goto UNREF;
  }

  uint32_t contentOffset = rn.segment * p->segmentLen;
  uint32_t contentLen = RTE_MIN(p->segmentLen, fd->lsL - contentOffset);
  struct iovec iov[LpMaxFragments];
  int iovcnt = 0;
  struct rte_mbuf* data =
    DataEnc_EncodeRoom(PName_ToLName(&pi->name), (LName){0}, fd->meta, contentLen, iov, &iovcnt,
                       &p->mp, Face_PacketTxAlign(p->face));
  if (unlikely(data == NULL)) {
    N_LOGW("Ls fd=%d drop=no-data", fd->fd);
    goto UNREF;
  }
  spdk_copy_buf_to_iovs(iov, iovcnt, RTE_PTR_ADD(fd->lsV, contentOffset), contentLen);
  FileServer_SignAndSend(p, ctx, fd, "Ls", data, *Packet_GetLpL3Hdr(interest));
  goto UNREF;

UNREF:
  FileServerFd_Unref(p, fd);
  NULLize(fd);
}

__attribute__((nonnull)) static void
FileServerRx_Metadata(FileServer* p, RxBurstCtx* ctx, FileServerRequestName rn) {
  ++p->cnt.reqMetadata;
  Packet* interest = Packet_FromMbuf(ctx->interest[ctx->index]);
  PInterest* pi = Packet_GetInterestHdr(interest);

  FileServerFd* fd = FileServerFd_Open(p, &pi->name, ctx->now);
  if (unlikely(fd == NULL)) {
    N_LOGD("Metadata drop=no-fd");
    return;
  }

  const uint8_t* metaInfo = NULL;
  uint32_t contentLen = 0;
  if (unlikely(fd == FileServer_NotFound)) {
    metaInfo = MetaInfo_Nack;
  } else if (unlikely((rn.kind & FileServerRequestLs) != 0 && !FileServerFd_IsDir(fd))) {
    metaInfo = MetaInfo_Nack;
  } else {
    metaInfo = MetaInfo_Metadata;
    contentLen = FileServerFd_PrepareMetadata(p, fd);
    NDNDPDK_ASSERT(contentLen > 0);
  }

  struct timespec utcNow;
  int res = clock_gettime(CLOCK_REALTIME, &utcNow);
  NDNDPDK_ASSERT(res == 0);
  uint8_t suffixV[20];
  LName suffix = (LName){.length = 0, .value = suffixV};
  suffix.length =
    Nni_EncodeNameComponent(suffixV, TtVersionNameComponent, FileServerFd_StatTime(utcNow));
  rte_memcpy(RTE_PTR_ADD(suffixV, suffix.length), NameComponent_Segment0,
             sizeof(NameComponent_Segment0));
  suffix.length += sizeof(NameComponent_Segment0);

  struct iovec iov[LpMaxFragments];
  int iovcnt = 0;
  struct rte_mbuf* data = DataEnc_EncodeRoom(PName_ToLName(&pi->name), suffix, metaInfo, contentLen,
                                             iov, &iovcnt, &p->mp, Face_PacketTxAlign(p->face));
  if (unlikely(data == NULL)) {
    N_LOGW("Metadata fd=%d drop=no-data", fd->fd);
    goto UNREF;
  }
  if (likely(contentLen > 0)) {
    FileServerFd_WriteMetadata(fd, iov, iovcnt);
  }
  FileServer_SignAndSend(p, ctx, fd, "Metadata", data, *Packet_GetLpL3Hdr(interest));
  goto UNREF;

UNREF:
  if (likely(fd != FileServer_NotFound)) {
    FileServerFd_Unref(p, fd);
  }
}

__attribute__((nonnull)) static inline void
FileServerRx_ProcessInterest(FileServer* p, RxBurstCtx* ctx) {
  struct rte_mbuf* interest = ctx->interest[ctx->index];
  Packet* npkt = Packet_FromMbuf(interest);
  PInterest* pi = Packet_GetInterestHdr(npkt);
  FileServerRequestName rn = FileServer_ParseRequest(pi);
  switch ((uint32_t)rn.kind) {
    case FileServerRequestVersion | FileServerRequestSegment:
      FileServerRx_Read(p, ctx, rn);
      break;
    case FileServerRequestLs | FileServerRequestVersion | FileServerRequestSegment:
      FileServerRx_Ls(p, ctx, rn);
      break;
    case FileServerRequestMetadata:
    case FileServerRequestLs | FileServerRequestMetadata:
      FileServerRx_Metadata(p, ctx, rn);
      break;
    default:
      N_LOGD("I drop=bad-name rn-kind=%" PRIx32, (uint32_t)rn.kind);
      break;
  }
}

uint32_t
FileServer_RxBurst(FileServer* p) {
  RxBurstCtx ctx;
  ctx.now = rte_get_tsc_cycles();
  ctx.index = 0;
  ctx.nData = 0;
  PktQueuePopResult pop = PktQueue_Pop(&p->rxQueue, ctx.interest, MaxBurstSize, ctx.now);
  if (unlikely(pop.count == 0)) {
    return 0;
  }

  for (ctx.nInterest = pop.count; ctx.index < ctx.nInterest; ++ctx.index) {
    FileServerRx_ProcessInterest(p, &ctx);
    // upon failure, callee sets ctx.index to ctx.nInterest, stopping the loop
  }

  Uring_Submit(&p->ur, p->uringWaitLbound, MaxBurstSize);
  Face_TxBurst(p->face, ctx.data, ctx.nData);
  rte_pktmbuf_free_bulk(ctx.interest, ctx.nInterest);
  return ctx.nInterest;
}
