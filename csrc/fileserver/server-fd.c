#include "server.h"

#include "../core/logger.h"
#include "naming.h"
#include <fcntl.h>
#include <sys/stat.h>
#include <sys/syscall.h>
#include <unistd.h>

N_LOG_INIT(FileServerFd);

#define uthash_malloc(sz) rte_malloc("FileServer.uthash", (sz), 0)
#define uthash_free(ptr, sz) rte_free((ptr))
#define HASH_KEYCMP(a, b, n) (!FdHt_Cmp_((const FileServerFd*)(a), (const LName*)(b)))
#define uthash_fatal(msg) rte_panic("uthash_fatal %s", msg)

#include "../vendor/uthash.h"

#undef HASH_INITIAL_NUM_BUCKETS
#undef HASH_INITIAL_NUM_BUCKETS_LOG2
#undef HASH_BKT_CAPACITY_THRESH
#undef HASH_EXPAND_BUCKETS
#define HASH_INITIAL_NUM_BUCKETS (p->nFdHtBuckets)
#define HASH_INITIAL_NUM_BUCKETS_LOG2 (rte_log2_u32(HASH_INITIAL_NUM_BUCKETS))
#define HASH_BKT_CAPACITY_THRESH UINT_MAX
#define HASH_EXPAND_BUCKETS(hh, tbl, oomed) FdHt_Expand_(tbl)

__attribute__((nonnull)) static inline bool
FdHt_Cmp_(const FileServerFd* entry, const LName* search)
{
  return entry->nameL == search->length && memcmp(entry->nameV, search->value, entry->nameL) == 0;
}

static __rte_noinline void
FdHt_Expand_(UT_hash_table* tbl)
{
  N_LOGE("FdHt Expand-rejected tbl=%p num_items=%u num_buckets=%u", tbl, tbl->num_items,
         tbl->num_buckets);
}

static FileServerFd notFound;
FileServerFd* FileServer_NotFound = &notFound;
static const unsigned StatxMask = STATX_TYPE | STATX_MODE | STATX_UID | STATX_GID | STATX_ATIME |
                                  STATX_MTIME | STATX_SIZE | STATX_BTIME;

FileServerFd*
FileServer_FdOpen(FileServer* p, const PName* name)
{
  LName prefix = PName_GetPrefix(name, name->firstNonGeneric);
  uint64_t hash = PName_ComputePrefixHash(name, name->firstNonGeneric);
  FileServerFd* entry = NULL;
  HASH_FIND_BYHASHVALUE(hh, p->fdHt, &prefix, 0, hash, entry);
  if (likely(entry != NULL)) {
    if (unlikely(entry->refcnt == 0)) {
      TAILQ_REMOVE(&p->fdQ, entry, queueNode);
      --p->fdQCount;
    }
    ++entry->refcnt;
    N_LOGD("FdOpen found fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
    return entry;
  }

  int mount = LNamePrefixFilter_Find(prefix, FileServerMaxMounts, p->mountPrefixL, p->mountPrefixV);
  if (unlikely(mount < 0)) {
    N_LOGD("FdOpen bad-name" N_LOG_ERROR("no-mount"));
    return NULL;
  }

  char filename[PATH_MAX];
  if (unlikely(!FileServer_ToFilename(name, p->mountPrefixComps[mount], filename))) {
    N_LOGD("FdOpen bad-name" N_LOG_ERROR("invalid-filename"));
    return NULL;
  }

  int fd = openat(p->dfd[mount], filename, O_RDONLY);
  if (unlikely(fd < 0)) {
    N_LOGD("FdOpen not-found" N_LOG_ERROR("openat errno=%d"), errno);
    return FileServer_NotFound;
  }

  struct rte_mbuf* mbuf = rte_pktmbuf_alloc(p->payloadMp);
  if (unlikely(mbuf == NULL)) {
    N_LOGE("FdOpen alloc-err" N_LOG_ERROR_BLANK);
    goto FAIL_FD;
  }

  static_assert(RTE_PKTMBUF_HEADROOM >= RTE_CACHE_LINE_SIZE, "");
  entry = RTE_PTR_ALIGN_FLOOR(mbuf->buf_addr, RTE_CACHE_LINE_SIZE);
  if (unlikely(syscall(__NR_statx, fd, "", FileServer_AT_EMPTY_PATH_, StatxMask, &entry->st) !=
               0)) {
    N_LOGD("FdOpen statx-error" N_LOG_ERROR("statx errno=%d"), errno);
    goto FAIL_MBUF;
  }

  entry->mbuf = mbuf;
  entry->fd = fd;
  entry->refcnt = 1;
  entry->nameL = prefix.length;
  rte_memcpy(entry->nameV, prefix.value, entry->nameL);

  entry->lastSeg = DIV_CEIL(entry->st.stx_size, p->segmentLen) - (uint64_t)(entry->st.stx_size > 0);
  uint8_t finalBlockV[10] = { TtSegmentNameComponent };
  finalBlockV[1] = Nni_Encode(&finalBlockV[2], entry->lastSeg);
  DataEnc_PrepareMetaInfo(&entry->meta, ContentBlob, 300000,
                          (LName){ .length = 2 + finalBlockV[1], .value = finalBlockV });

  HASH_ADD_BYHASHVALUE(hh, p->fdHt, self, 0, hash, entry);
  N_LOGD("FdOpen open fd=%d mount=%d filename=%s", entry->fd, mount, filename);
  return entry;

FAIL_MBUF:
  rte_pktmbuf_free(mbuf);
FAIL_FD:
  close(fd);
  return NULL;
}

void
FileServer_FdUnref(FileServer* p, FileServerFd* entry)
{
  --entry->refcnt;
  if (likely(entry->refcnt > 0)) {
    N_LOGD("FdUnref in-use fd=%d refcnt=%d", entry->fd, entry->refcnt);
    return;
  }

  N_LOGD("FdUnref keep fd=%d", entry->fd);
  TAILQ_INSERT_TAIL(&p->fdQ, entry, queueNode);
  ++p->fdQCount;
  if (unlikely(p->fdQCount <= p->fdQCapacity)) {
    return;
  }

  FileServerFd* evict = TAILQ_FIRST(&p->fdQ);
  N_LOGD("FdUnref close fd=%d", evict->fd);
  HASH_DELETE(hh, p->fdHt, evict);
  TAILQ_REMOVE(&p->fdQ, evict, queueNode);
  --p->fdQCount;
  close(evict->fd);
  rte_pktmbuf_free(evict->mbuf);
}

void
FileServer_FdClear(FileServer* p)
{
  FileServerFd* entry;
  FileServerFd* tmp;
  HASH_ITER (hh, p->fdHt, entry, tmp) {
    N_LOGD("FdClear close fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
    close(entry->fd);
    rte_pktmbuf_free(entry->mbuf);
  }
  HASH_CLEAR(hh, p->fdHt);
  TAILQ_INIT(&p->fdQ);
  p->fdQCount = 0;
}
