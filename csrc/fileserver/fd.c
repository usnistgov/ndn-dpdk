#include "fd.h"
#include "../core/logger.h"
#include "../ndni/tlv-encoder.h"
#include "naming.h"
#include "server.h"
#include <dirent.h>
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
  return entry->prefixL == search->length &&
         memcmp(entry->nameV, search->value, search->length) == 0;
}

static __rte_noinline void
FdHt_Expand_(UT_hash_table* tbl)
{
  N_LOGE("FdHt Expand-rejected tbl=%p num_items=%u num_buckets=%u", tbl, tbl->num_items,
         tbl->num_buckets);
}

enum
{
  /// Maximum mount+path TLV-LENGTH to accommodate [32=ls]+version+segment components.
  FileServer_MaxPrefixL = NameMaxLength - sizeof(FileServer_KeywordLs) - 10 - 10,
};

static FileServerFd notFound;
FileServerFd* FileServer_NotFound = &notFound;

__attribute__((nonnull)) static inline int
FileServerFd_UpdateStatx(FileServer* p, FileServerFd* entry, TscTime now, bool* changed)
{
  *changed = true;
  uint64_t oldVersion = entry->version;
  uint64_t oldSize = entry->st.stx_size;

  int res = syscall(__NR_statx, entry->fd, "", AT_EMPTY_PATH,
                    FileServerStatxRequired | FileServerStatxOptional, &entry->st);
  if (likely(res == 0)) {
    if (unlikely(!FileServerFd_HasStatBit(entry, FileServerStatxRequired) ||
                 !(FileServerFd_IsFile(entry) || FileServerFd_IsDir(entry)))) {
      res = EPIPE; // use an "impossible" errno to indicate this condition
    } else {
      entry->version = FileServerFd_StatTime(entry->st.stx_mtime);
      *changed = oldVersion != entry->version || oldSize != entry->st.stx_size;
    }
  }
  static_assert(sizeof(entry->st.stx_ino) == sizeof(TscTime), "");
  entry->st.stx_ino = now + p->statValidity;
  return res;
}

__attribute__((nonnull)) static inline void
FileServerFd_PrepapeMeta(FileServer* p, FileServerFd* entry)
{
  entry->lastSeg = DIV_CEIL(entry->st.stx_size, p->segmentLen) - (uint64_t)(entry->st.stx_size > 0);

  uint16_t nameL = entry->prefixL;
  if (unlikely(FileServerFd_IsDir(entry))) {
    rte_memcpy(RTE_PTR_ADD(entry->nameV, nameL), FileServer_KeywordLs,
               sizeof(FileServer_KeywordLs));
    nameL += sizeof(FileServer_KeywordLs);
  }

  uint8_t* version = RTE_PTR_ADD(entry->nameV, nameL);
  version[0] = TtVersionNameComponent;
  version[1] = Nni_Encode(&version[2], entry->version);
  entry->versionedL = (nameL += 2 + version[1]);

  uint8_t* segment = RTE_PTR_ADD(entry->nameV, nameL);
  segment[0] = TtSegmentNameComponent;
  segment[1] = Nni_Encode(&segment[2], entry->lastSeg);
  entry->segmentL = (nameL += 2 + segment[1]);

  DataEnc_MustPrepareMetaInfo(&entry->meta, ContentBlob, 0,
                              ((LName){ .length = 2 + segment[1], .value = segment }));
}

__attribute__((nonnull)) static inline FileServerFd*
FileServerFd_Ref(FileServer* p, FileServerFd* entry, TscTime now)
{
  if (unlikely(entry->refcnt == 0)) {
    TAILQ_REMOVE(&p->fdQ, entry, queueNode);
    --p->fdQCount;
  }

  if (unlikely((TscTime)entry->st.stx_ino < now)) {
    bool changed = false;
    int res = FileServerFd_UpdateStatx(p, entry, now, &changed);
    if (unlikely(res != 0)) {
      N_LOGD(
        "Ref statx-update fd=%d refcnt=%" PRIu16 N_LOG_ERROR("statx-res=%d statx-mask=0x%" PRIx32),
        entry->fd, entry->refcnt, res, entry->st.stx_mask);
      return NULL;
    }
    N_LOGD("Ref statx-update fd=%d refcnt=%" PRIu16 " version=%" PRIu64 " size=%" PRIu64
           " changed=%d",
           entry->fd, entry->refcnt, entry->version, (uint64_t)entry->st.stx_size, (int)changed);
    FileServerFd_PrepapeMeta(p, entry);
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

  int fd = -1;
  if (likely(filename[0] != '\0')) {
    fd = openat(p->dfd[mount], filename, O_RDONLY);
    if (unlikely(fd < 0)) {
      N_LOGD("New openat-err mount=%d filename=%s" N_LOG_ERROR("errno=%d"), mount, filename, errno);
      return FileServer_NotFound;
    }
  } else {
    fd = dup(p->dfd[mount]);
    if (unlikely(fd < 0)) {
      N_LOGD("New dup-err mount=%d filename=''" N_LOG_ERROR("errno=%d"), mount, errno);
      return FileServer_NotFound;
    }
  }

  struct rte_mbuf* mbuf = rte_pktmbuf_alloc(p->payloadMp);
  if (unlikely(mbuf == NULL)) {
    N_LOGE("New mbuf-alloc-err" N_LOG_ERROR_BLANK);
    goto FAIL_FD;
  }

  static_assert(RTE_PKTMBUF_HEADROOM >= RTE_CACHE_LINE_SIZE, "");
  FileServerFd* entry = RTE_PTR_ALIGN_FLOOR(mbuf->buf_addr, RTE_CACHE_LINE_SIZE);
  entry->fd = fd;

  entry->st.stx_size = 0;
  bool changed_ = false;
  int res = FileServerFd_UpdateStatx(p, entry, now, &changed_);
  if (unlikely(res != 0)) {
    N_LOGD("New mount=%d filename=%s" N_LOG_ERROR("statx-res=%d statx-mask=0x%" PRIx32), mount,
           filename, res, entry->st.stx_mask);
    goto FAIL_MBUF;
  }

  entry->mbuf = mbuf;
  entry->refcnt = 1;
  entry->prefixL = prefix.length;
  rte_memcpy(entry->nameV, prefix.value, prefix.length);
  FileServerFd_PrepapeMeta(p, entry);

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
  LName prefix = FileServer_GetPrefix(name);
  if (unlikely(prefix.length > FileServer_MaxPrefixL)) {
    return NULL;
  }
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

bool
FileServerFd_EncodeMetadata(FileServer* p, FileServerFd* entry, struct rte_mbuf* payload)
{
  if (unlikely(rte_pktmbuf_tailroom(payload) <
               entry->versionedL + FileServerEstimatedMetadataSize)) {
    return false;
  }
  uint8_t* value = rte_pktmbuf_mtod(payload, uint8_t*);
  size_t off = 0;

#define HAS_STAT_BIT(bit)                                                                          \
  (likely((FileServerStatxRequired & (bit)) == (bit) || FileServerFd_HasStatBit(entry, (bit))))

#define APPEND_NNI(type, bits, val)                                                                \
  do {                                                                                             \
    struct                                                                                         \
    {                                                                                              \
      unaligned_uint32_t tl;                                                                       \
      unaligned_uint##bits##_t v;                                                                  \
    } __rte_packed* f = RTE_PTR_ADD(value, off);                                                   \
    f->tl = TlvEncoder_ConstTL3(TtFileServer##type, sizeof(f->v));                                 \
    f->v = rte_cpu_to_be_##bits((uint##bits##_t)(val));                                            \
    off += sizeof(*f);                                                                             \
  } while (false)

  value[off++] = TtName;
  off += TlvEncoder_WriteVarNum(&value[off], entry->versionedL);
  rte_memcpy(&value[off], entry->nameV, entry->versionedL);
  off += entry->versionedL;

  if (likely(FileServerFd_IsFile(entry))) {
    NDNDPDK_ASSERT(entry->meta.value[2] == TtFinalBlock);
    rte_memcpy(&value[off], &entry->meta.value[2], entry->meta.value[1]);
    off += entry->meta.value[1];
    APPEND_NNI(SegmentSize, 16, p->segmentLen);
    if (HAS_STAT_BIT(STATX_SIZE)) {
      APPEND_NNI(Size, 64, entry->st.stx_size);
    }
  }
  if (HAS_STAT_BIT(STATX_TYPE | STATX_MODE)) {
    APPEND_NNI(Mode, 16, entry->st.stx_mode);
  }
  if (HAS_STAT_BIT(STATX_ATIME)) {
    APPEND_NNI(Atime, 64, FileServerFd_StatTime(entry->st.stx_atime));
  }
  if (HAS_STAT_BIT(STATX_BTIME)) {
    APPEND_NNI(Btime, 64, FileServerFd_StatTime(entry->st.stx_btime));
  }
  if (HAS_STAT_BIT(STATX_CTIME)) {
    APPEND_NNI(Ctime, 64, FileServerFd_StatTime(entry->st.stx_ctime));
  }
  if (HAS_STAT_BIT(STATX_MTIME)) {
    APPEND_NNI(Mtime, 64, entry->version);
  }

#undef APPEND_NNI
#undef HAS_STAT_BIT
  const char* room = rte_pktmbuf_append(payload, off);
  NDNDPDK_ASSERT(room != NULL);
  return true;
}

bool
FileServerFd_EncodeLs(FileServer* p, FileServerFd* entry, struct rte_mbuf* payload,
                      uint16_t segmentLen)
{
  NDNDPDK_ASSERT(FileServerFd_IsDir(entry));
  int dfd = dup(entry->fd);
  if (unlikely(dfd < 0)) {
    N_LOGD("Ls dup-err fd=%d errno=%d" N_LOG_ERROR_BLANK, entry->fd, errno);
    return false;
  }

  DIR* dir = fdopendir(dfd);
  if (unlikely(dir == NULL)) {
    N_LOGD("Ls fdopendir-err fd=%d dfd=%d errno=%d" N_LOG_ERROR_BLANK, entry->fd, dfd, errno);
    close(dfd);
    return false;
  }

  char* value = rte_pktmbuf_mtod(payload, char*);
  size_t off = 0;
  struct dirent* ent = NULL;
  while ((ent = readdir(dir)) != NULL) {
    bool isDir = false;
    switch (ent->d_type) {
      case DT_REG:
        break;
      case DT_DIR:
        isDir = 1;
        break;
      default:
        continue;
    }

    size_t entNameL = strlen(ent->d_name);
    if (unlikely(entNameL <= 2 && strspn(ent->d_name, ".") == entNameL)) { // . or ..
      continue;
    }
    uint16_t lineL = entNameL + (uint16_t)isDir + 1;
    if (unlikely(off + lineL > segmentLen)) {
      break;
    }
    rte_memcpy(&value[off], ent->d_name, entNameL);
    off += entNameL;
    if (isDir) {
      value[off++] = '/';
    }
    value[off++] = '\0';
  }

  closedir(dir);
  const char* room = rte_pktmbuf_append(payload, off);
  NDNDPDK_ASSERT(room != NULL);
  return true;
}
