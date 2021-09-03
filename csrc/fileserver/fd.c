#include "fd.h"
#include "../core/logger.h"
#include "naming.h"
#include "server.h"
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

__attribute__((nonnull)) static inline int
FileServerFd_UpdateStatx(FileServer* p, FileServerFd* entry, TscTime now)
{
  int res = syscall(__NR_statx, entry->fd, "", AT_EMPTY_PATH,
                    FileServerStatxRequired | FileServerStatxOptional, &entry->st);
  if (likely(res == 0 || FileServerFd_HasStatBit(entry, FileServerStatxRequired))) {
    static_assert(sizeof(entry->st.stx_ino) == sizeof(TscTime), "");
    entry->st.stx_ino = now + p->statValidity;

    entry->lastSeg =
      DIV_CEIL(entry->st.stx_size, p->segmentLen) - (uint64_t)(entry->st.stx_size > 0);
    uint8_t finalBlockV[10] = { TtSegmentNameComponent };
    finalBlockV[1] = Nni_Encode(&finalBlockV[2], entry->lastSeg);
    DataEnc_MustPrepareMetaInfo(&entry->meta, ContentBlob, 0,
                                ((LName){ .length = 2 + finalBlockV[1], .value = finalBlockV }));
  }
  return res;
}

__attribute__((nonnull)) static inline FileServerFd*
FileServerFd_Ref(FileServer* p, FileServerFd* entry, TscTime now)
{
  if (unlikely(entry->refcnt == 0)) {
    TAILQ_REMOVE(&p->fdQ, entry, queueNode);
    --p->fdQCount;
  }

  if (unlikely((TscTime)entry->st.stx_ino < now)) {
    uint64_t oldSize = entry->st.stx_size;
    int res = FileServerFd_UpdateStatx(p, entry, now);
    if (unlikely(res != 0)) {
      N_LOGD(
        "Ref statx-update fd=%d refcnt=%" PRIu16 N_LOG_ERROR("statx-res=%d statx-mask=0x%" PRIx32),
        entry->fd, entry->refcnt, res, entry->st.stx_mask);
      return NULL;
    }
    N_LOGD("Ref statx-update fd=%d refcnt=%" PRIu16 " size=%" PRIu64 " size-changed=%d", entry->fd,
           entry->refcnt, (uint64_t)entry->st.stx_size, (int)(oldSize != entry->st.stx_size));
  }

  ++entry->refcnt;
  N_LOGD("Ref fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
  return entry;
}

__attribute__((nonnull)) static FileServerFd*
FileServerFd_New(FileServer* p, const PName* name, LName prefix, uint64_t hash, TscTime now)
{
  int mount = LNamePrefixFilter_Find(prefix, FileServerMaxMounts, p->mountPrefixL, p->mountPrefixV);
  if (unlikely(mount < 0)) {
    N_LOGD("New bad-name" N_LOG_ERROR("mount-not-matched"));
    return NULL;
  }

  char filename[PATH_MAX];
  if (unlikely(!FileServer_ToFilename(name, p->mountPrefixComps[mount], filename))) {
    N_LOGD("New bad-name" N_LOG_ERROR("invalid-filename"));
    return NULL;
  }

  int fd = openat(p->dfd[mount], filename, O_RDONLY);
  if (unlikely(fd < 0)) {
    N_LOGD("New openat-err mount=%d filename=%s" N_LOG_ERROR("errno=%d"), mount, filename, errno);
    return FileServer_NotFound;
  }

  struct rte_mbuf* mbuf = rte_pktmbuf_alloc(p->payloadMp);
  if (unlikely(mbuf == NULL)) {
    N_LOGE("New mbuf-alloc-err" N_LOG_ERROR_BLANK);
    goto FAIL_FD;
  }

  static_assert(RTE_PKTMBUF_HEADROOM >= RTE_CACHE_LINE_SIZE, "");
  FileServerFd* entry = RTE_PTR_ALIGN_FLOOR(mbuf->buf_addr, RTE_CACHE_LINE_SIZE);
  entry->fd = fd;
  int res = FileServerFd_UpdateStatx(p, entry, now);
  if (unlikely(res != 0)) {
    N_LOGD("New mount=%d filename=%s" N_LOG_ERROR("statx-res=%d statx-mask=0x%" PRIx32), mount,
           filename, res, entry->st.stx_mask);
    goto FAIL_MBUF;
  }

  entry->mbuf = mbuf;
  entry->refcnt = 1;
  entry->nameL = prefix.length;
  rte_memcpy(entry->nameV, prefix.value, entry->nameL);

  HASH_ADD_BYHASHVALUE(hh, p->fdHt, self, 0, hash, entry);
  N_LOGD("New mount=%d filename=%s fd=%d statx-mask=0x%" PRIu32 " size=%" PRIu64, mount, filename,
         entry->fd, entry->st.stx_mask, (uint64_t)entry->st.stx_size);
  return entry;

FAIL_MBUF:
  rte_pktmbuf_free(mbuf);
FAIL_FD:
  close(fd);
  return NULL;
}

FileServerFd*
FileServerFd_Open(FileServer* p, const PName* name, TscTime now)
{
  LName prefix = PName_GetPrefix(name, name->firstNonGeneric);
  uint64_t hash = PName_ComputePrefixHash(name, name->firstNonGeneric);
  FileServerFd* entry = NULL;
  HASH_FIND_BYHASHVALUE(hh, p->fdHt, &prefix, 0, hash, entry);
  if (likely(entry != NULL)) {
    return FileServerFd_Ref(p, entry, now);
  }
  return FileServerFd_New(p, name, prefix, hash, now);
}

void
FileServerFd_Unref(FileServer* p, FileServerFd* entry)
{
  --entry->refcnt;
  if (likely(entry->refcnt > 0)) {
    N_LOGD("Unref in-use fd=%d refcnt=%d", entry->fd, entry->refcnt);
    return;
  }

  N_LOGD("Unref keep fd=%d", entry->fd);
  TAILQ_INSERT_TAIL(&p->fdQ, entry, queueNode);
  ++p->fdQCount;
  if (unlikely(p->fdQCount <= p->fdQCapacity)) {
    return;
  }

  FileServerFd* evict = TAILQ_FIRST(&p->fdQ);
  N_LOGD("Unref close fd=%d", evict->fd);
  HASH_DELETE(hh, p->fdHt, evict);
  TAILQ_REMOVE(&p->fdQ, evict, queueNode);
  --p->fdQCount;
  close(evict->fd);
  rte_pktmbuf_free(evict->mbuf);
}

void
FileServerFd_Clear(FileServer* p)
{
  FileServerFd* entry;
  FileServerFd* tmp;
  HASH_ITER (hh, p->fdHt, entry, tmp) {
    N_LOGD("Clear close fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
    close(entry->fd);
    rte_pktmbuf_free(entry->mbuf);
  }
  HASH_CLEAR(hh, p->fdHt);
  TAILQ_INIT(&p->fdQ);
  p->fdQCount = 0;
}
