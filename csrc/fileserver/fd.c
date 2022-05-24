#include "fd.h"
#include "../core/logger.h"
#include "../ndni/tlv-encoder.h"
#include "an.h"
#include "naming.h"
#include "server.h"
#include <dirent.h>
#include <sys/syscall.h>
#include <unistd.h>

N_LOG_INIT(FileServerFd);

#define uthash_malloc(sz) rte_malloc("FileServer.uthash", (sz), 0)
#define uthash_free(ptr, sz) rte_free((ptr))
#define HASH_FUNCTION HASH_FUNCTION_DONOTUSE
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

static FileServerFd notFound;
FileServerFd* FileServer_NotFound = &notFound;

/** @brief Reuse FileServerFd.st.stx_ino field as (TscTime)nextUpdate. */
#define FdStx_NextUpdate stx_ino
static_assert(RTE_SIZEOF_FIELD(struct statx, FdStx_NextUpdate) == sizeof(TscTime), "");

enum
{
  FileServerStatxRequired = STATX_TYPE | STATX_MODE | STATX_MTIME | STATX_SIZE,
  FileServerStatxOptional = STATX_ATIME | STATX_CTIME | STATX_BTIME,
};

static __rte_always_inline bool
FileServerFd_HasStatBit(const FileServerFd* entry, uint32_t bit)
{
  return (entry->st.stx_mask & bit) == bit;
}

static __rte_always_inline uint64_t
FileServerFd_StatTime(struct statx_timestamp t)
{
  return (uint64_t)t.tv_sec * SPDK_SEC_TO_NSEC + (uint64_t)t.tv_nsec;
}

__attribute__((nonnull)) static inline int
FileServerFd_UpdateStatx(FileServer* p, FileServerFd* entry, TscTime now, bool* changed)
{
  *changed = true;
  uint64_t oldVersion = entry->version;
  uint64_t oldSize = entry->st.stx_size;

  int res = syscall(__NR_statx, entry->fd, "", AT_EMPTY_PATH,
                    FileServerStatxRequired | FileServerStatxOptional, &entry->st);
  if (likely(res == 0)) {
    if (likely(FileServerFd_HasStatBit(entry, FileServerStatxRequired) &&
               (FileServerFd_IsFile(entry) || FileServerFd_IsDir(entry)))) {
      entry->version = FileServerFd_StatTime(entry->st.stx_mtime);
      *changed = oldVersion != entry->version || oldSize != entry->st.stx_size;
    } else {
      res = ETXTBSY; // use an "impossible" errno to indicate statx error condition
    }
  }
  entry->st.FdStx_NextUpdate = now + p->statValidity;
  return res;
}

__attribute__((nonnull)) static inline void
FileServerFd_PrepapeVersionedName(FileServer* p, FileServerFd* entry)
{
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
}

__attribute__((nonnull)) static inline void
FileServerFd_PrepapeMetaInfo(FileServer* p, FileServerFd* entry, uint64_t size)
{
  entry->lastSeg = SPDK_CEIL_DIV(size, p->segmentLen) - (uint64_t)(size > 0);

  uint8_t segment[10];
  segment[0] = TtSegmentNameComponent;
  segment[1] = Nni_Encode(&segment[2], entry->lastSeg);
  DataEnc_PrepareMetaInfo(&entry->meta, ContentBlob, 0,
                          ((LName){ .length = 2 + segment[1], .value = segment }));
}

__attribute__((nonnull)) static inline FileServerFd*
FileServerFd_Ref(FileServer* p, FileServerFd* entry, TscTime now)
{
  if (unlikely(entry->refcnt == 0)) {
    cds_list_del(&entry->queueNode);
    --p->fdQCount;
  }

  if (unlikely((TscTime)entry->st.FdStx_NextUpdate < now)) {
    bool changed = false;
    int res = FileServerFd_UpdateStatx(p, entry, now, &changed);
    ++p->cnt.fdUpdateStat;
    if (unlikely(res != 0)) {
      N_LOGD(
        "Ref statx-update fd=%d refcnt=%" PRIu16 N_LOG_ERROR("statx-res=%d statx-mask=0x%" PRIx32),
        entry->fd, entry->refcnt, res, entry->st.stx_mask);
      return NULL;
    }
    N_LOGD("Ref statx-update fd=%d refcnt=%" PRIu16 " version=%" PRIu64 " size=%" PRIu64
           " changed=%d",
           entry->fd, entry->refcnt, entry->version, (uint64_t)entry->st.stx_size, (int)changed);
    if (changed) {
      FileServerFd_PrepapeVersionedName(p, entry);
      FileServerFd_PrepapeMetaInfo(p, entry, entry->st.stx_size);
      entry->lsL = UINT32_MAX;
    }
  }

  ++entry->refcnt;
  N_LOGD("Ref fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
  return entry;
}

__attribute__((nonnull)) static FileServerFd*
FileServerFd_New(FileServer* p, const PName* name, LName prefix, uint64_t hash, TscTime now)
{
  FileServerFd* errEntry = NULL;
  int mount = LNamePrefixFilter_Find(prefix, FileServerMaxMounts, p->mountPrefixL, p->mountPrefixV);
  if (unlikely(mount < 0)) {
    N_LOGD("New bad-name" N_LOG_ERROR("mount-not-matched"));
    goto FAIL_OUT;
  }

  char filename[PATH_MAX];
  if (unlikely(!FileServer_ToFilename(name, p->mountPrefixComps[mount], filename))) {
    N_LOGD("New bad-name" N_LOG_ERROR("invalid-filename"));
    goto FAIL_OUT;
  }

  errEntry = FileServer_NotFound;
  int fd = -1;
  const char* logFilename = NULL;
  if (likely(filename[0] != '\0')) {
    logFilename = filename;
    fd = openat(p->dfd[mount], filename, O_RDONLY);
    if (unlikely(fd < 0)) {
      ++p->cnt.fdNotFound;
      N_LOGD("New openat-err mount=%d filename=%s" N_LOG_ERROR_ERRNO, mount, filename, errno);
      goto FAIL_OUT;
    }
  } else {
    logFilename = "(empty)";
    fd = dup(p->dfd[mount]);
    if (unlikely(fd < 0)) {
      ++p->cnt.fdNotFound;
      N_LOGD("New dup-err mount=%d filename=''" N_LOG_ERROR_ERRNO, mount, errno);
      errEntry = NULL;
      goto FAIL_OUT;
    }
  }
  ++p->cnt.fdNew;

  FileServerFd* entry = NULL;
  int res = rte_mempool_get(p->fdMp, (void**)&entry);
  if (unlikely(res != 0)) {
    N_LOGE("New fd-alloc-err" N_LOG_ERROR_BLANK);
    errEntry = NULL;
    goto FAIL_CLOSE;
  }
  entry->fd = fd;

  entry->version = 0;
  entry->st.stx_size = 0;
  bool changed_ = false;
  res = FileServerFd_UpdateStatx(p, entry, now, &changed_);
  if (unlikely(res != 0)) {
    N_LOGD("New mount=%d filename=%s" N_LOG_ERROR("statx-res=%d statx-mask=0x%" PRIx32), mount,
           logFilename, res, entry->st.stx_mask);
    goto FAIL_ALLOC;
  }

  entry->refcnt = 1;
  entry->lsL = UINT32_MAX;
  entry->prefixL = prefix.length;
  rte_memcpy(entry->nameV, prefix.value, prefix.length);
  FileServerFd_PrepapeVersionedName(p, entry);
  FileServerFd_PrepapeMetaInfo(p, entry, entry->st.stx_size);

  HASH_ADD_BYHASHVALUE(hh, p->fdHt, self, 0, hash, entry);
  N_LOGD("New mount=%d filename=%s fd=%d statx-mask=0x%" PRIu32 " version=%" PRIu64
         " size=%" PRIu64,
         mount, logFilename, entry->fd, entry->st.stx_mask, entry->version,
         (uint64_t)entry->st.stx_size);
  return entry;

FAIL_ALLOC:
  rte_mempool_put(p->fdMp, entry);
FAIL_CLOSE:
  close(fd);
FAIL_OUT:
  return errEntry;
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
  cds_list_add_tail(&entry->queueNode, &p->fdQ);
  ++p->fdQCount;
  NULLize(entry);
  if (unlikely(p->fdQCount <= p->fdQCapacity)) {
    return;
  }

  FileServerFd* evict = cds_list_first_entry(&p->fdQ, FileServerFd, queueNode);
  N_LOGD("Unref close fd=%d", evict->fd);
  HASH_DELETE(hh, p->fdHt, evict);
  cds_list_del(&evict->queueNode);
  --p->fdQCount;
  close(evict->fd);
  rte_mempool_put(p->fdMp, evict);
  ++p->cnt.fdClose;
}

void
FileServerFd_Clear(FileServer* p)
{
  FileServerFd* entry;
  FileServerFd* tmp;
  HASH_ITER (hh, p->fdHt, entry, tmp) {
    N_LOGD("Clear close fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
    close(entry->fd);
    rte_mempool_put(p->fdMp, entry);
  }
  HASH_CLEAR(hh, p->fdHt);
  CDS_INIT_LIST_HEAD(&p->fdQ);
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
    f->tl = TlvEncoder_ConstTL3(TtFile##type, sizeof(f->v));                                       \
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

__attribute__((nonnull)) static inline int
FileServerFd_DirentType(FileServerFd* entry, struct dirent64* de)
{
  switch (de->d_type) {
    case DT_UNKNOWN:
    case DT_LNK:
      break;
    default:
      return de->d_type;
  }

  struct statx st;
  int res = syscall(__NR_statx, entry->fd, de->d_name, 0, STATX_TYPE, &st);
  if (unlikely(res < 0 || (st.stx_mask & STATX_TYPE) == 0)) {
    return DT_UNKNOWN;
  }

  if (S_ISREG(st.stx_mode)) {
    return DT_REG;
  }
  if (S_ISDIR(st.stx_mode)) {
    return DT_DIR;
  }
  return DT_UNKNOWN;
}

bool
FileServerFd_GenerateLs(FileServer* p, FileServerFd* entry)
{
  NDNDPDK_ASSERT(FileServerFd_IsDir(entry));

  int res = lseek(entry->fd, 0, SEEK_SET);
  if (unlikely(res < 0)) {
    N_LOGD("Ls lseek-err fd=%d" N_LOG_ERROR_ERRNO, entry->fd, errno);
    goto FAIL;
  }

  entry->lsL = 0;
  bool isFull = false;
  uint8_t dents[spdk_min(FileServerMaxLsResult, 1 << 18)];
  while ((res = syscall(SYS_getdents64, entry->fd, dents, sizeof(dents))) > 0) {
    for (int offset = 0; offset < res;) {
      struct dirent64* de = RTE_PTR_ADD(dents, offset);
      offset += de->d_reclen;
      if (unlikely(de->d_ino == 0)) {
        continue;
      }

      uint16_t nameL = strnlen(de->d_name, RTE_DIM(de->d_name));
      if (unlikely(entry->lsL + nameL + 2 >= sizeof(entry->lsV))) {
        isFull = true;
        goto FULL;
      }
      if (unlikely(nameL <= 2 && nameL == strspn(de->d_name, "."))) {
        continue;
      }

      bool isDir = false;
      switch (FileServerFd_DirentType(entry, de)) {
        case DT_REG:
          break;
        case DT_DIR:
          isDir = true;
          break;
        default:
          continue;
      }

      rte_memcpy(&entry->lsV[entry->lsL], de->d_name, nameL);
      entry->lsL += nameL;
      if (isDir) {
        entry->lsV[entry->lsL++] = '/';
      }
      entry->lsV[entry->lsL++] = '\0';
    }
  }
  if (unlikely(res < 0)) {
    N_LOGD("Ls getdents64-err fd=%d" N_LOG_ERROR_ERRNO, entry->fd, errno);
    goto FAIL;
  }

FULL:
  FileServerFd_PrepapeMetaInfo(p, entry, entry->lsL);
  N_LOGD("Ls generated fd=%d version=%" PRIu64 " length=%" PRIu32 " full=%d", entry->fd,
         entry->version, entry->lsL, (int)isFull);
  return true;

FAIL:
  entry->lsL = UINT32_MAX;
  return false;
}
