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
FdHt_Cmp_(const FileServerFd* entry, const LName* search) {
  return entry->prefixL == search->length &&
         memcmp(entry->nameV, search->value, search->length) == 0;
}

static __rte_noinline void
FdHt_Expand_(UT_hash_table* tbl) {
  N_LOGE("FdHt Expand-rejected tbl=%p num_items=%u num_buckets=%u", tbl, tbl->num_items,
         tbl->num_buckets);
}

static FileServerFd notFound;
FileServerFd* FileServer_NotFound = &notFound;

/** @brief Reuse FileServerFd.st.stx_ino field as (TscTime)nextUpdate. */
#define FdStx_NextUpdate stx_ino
static_assert(RTE_SIZEOF_FIELD(struct statx, FdStx_NextUpdate) == sizeof(TscTime), "");

enum {
  FileServerStatxRequired = STATX_TYPE | STATX_MODE | STATX_MTIME | STATX_SIZE,
  FileServerStatxOptional = STATX_ATIME | STATX_CTIME | STATX_BTIME,
};

static __rte_always_inline bool
FileServerFd_HasStatBit(const FileServerFd* entry, uint32_t bit) {
  return (entry->st.stx_mask & bit) == bit;
}

static __rte_always_inline int
FileServerFd_InvokeStatx(FileServer* p, FileServerFd* entry, int dfd, const char* restrict pathname,
                         TscTime now) {
  int res = statx(dfd, pathname, AT_EMPTY_PATH, FileServerStatxRequired | FileServerStatxOptional,
                  &entry->st);
  entry->st.FdStx_NextUpdate = now + p->statValidity;
  if (unlikely(res != 0)) {
    return errno;
  }
  if (unlikely(!FileServerFd_HasStatBit(entry, FileServerStatxRequired))) {
    return ENOMSG;
  }
  if (unlikely(!FileServerFd_IsFile(entry) && !FileServerFd_IsDir(entry))) {
    return ENOTDIR;
  }
  entry->version = FileServerFd_StatTime(entry->st.stx_mtime);
  return 0;
}

__attribute__((nonnull)) static inline void
FileServerFd_PrepareVersionedName(FileServer* p, FileServerFd* entry) {
  uint16_t nameL = entry->prefixL;
  if (unlikely(FileServerFd_IsDir(entry))) {
    rte_memcpy(RTE_PTR_ADD(entry->nameV, nameL), FileServer_KeywordLs,
               sizeof(FileServer_KeywordLs));
    nameL += sizeof(FileServer_KeywordLs);
  }

  uint8_t* version = RTE_PTR_ADD(entry->nameV, nameL);
  uint8_t sizeofVersion = Nni_EncodeNameComponent(version, TtVersionNameComponent, entry->version);
  entry->versionedL = (nameL += sizeofVersion);
}

__attribute__((nonnull)) static inline void
FileServerFd_PrepareMetaInfo(FileServer* p, FileServerFd* entry, uint64_t size) {
  entry->lastSeg = SPDK_CEIL_DIV(size, p->segmentLen) - (uint64_t)(size > 0);

  uint8_t segment[10];
  LName finalBlock = (LName){
    .length = Nni_EncodeNameComponent(segment, TtSegmentNameComponent, entry->lastSeg),
    .value = segment,
  };
  DataEnc_PrepareMetaInfo(entry->meta, ContentBlob, 0, finalBlock);
}

__attribute__((nonnull)) static inline FileServerFd*
FileServerFd_Ref(FileServer* p, FileServerFd* entry, TscTime now) {
  if (unlikely(entry->refcnt == 0)) {
    cds_list_del(&entry->queueNode);
    --p->fdQCount;
  }

  if (unlikely((TscTime)entry->st.FdStx_NextUpdate < now)) {
    uint64_t oldVersion = entry->version;
    uint64_t oldSize = entry->st.stx_size;
    int res = FileServerFd_InvokeStatx(p, entry, entry->fd, "", now);
    ++p->cnt.fdUpdateStat;
    if (unlikely(res != 0)) {
      N_LOGD("Ref statx-err fd=%d refcnt=%" PRIu16 N_LOG_ERROR_ERRNO, entry->fd, entry->refcnt,
             res);
      return NULL;
    }

    bool changed = oldVersion != entry->version || oldSize != entry->st.stx_size;
    N_LOGD("Ref statx-update fd=%d refcnt=%" PRIu16 " version=%" PRIu64 " size=%" PRIu64
           " changed=%d",
           entry->fd, entry->refcnt, entry->version, (uint64_t)entry->st.stx_size, (int)changed);
    if (changed) {
      FileServerFd_PrepareVersionedName(p, entry);
      FileServerFd_PrepareMetaInfo(p, entry, entry->st.stx_size);
      entry->metadataL = 0;
      entry->lsL = UINT32_MAX;
    }
  }

  ++entry->refcnt;
  N_LOGD("Ref fd=%d refcnt=%" PRIu16, entry->fd, entry->refcnt);
  return entry;
}

__attribute__((nonnull)) static FileServerFd*
FileServerFd_New(FileServer* p, const PName* name, LName prefix, uint64_t hash, TscTime now) {
  int mount = LNamePrefixFilter_Find(prefix, FileServerMaxMounts, p->mountPrefixL, p->mountPrefixV);
  if (unlikely(mount < 0)) {
    N_LOGD("New bad-name" N_LOG_ERROR("mount-not-matched"));
    return NULL;
  }
  int dfd = p->dfd[mount];

  char filename[PATH_MAX];
  if (unlikely(!FileServer_ToFilename(name, p->mountPrefixComps[mount], filename))) {
    N_LOGD("New bad-name" N_LOG_ERROR("invalid-filename"));
    return NULL;
  }

  FileServerFd* entry = NULL;
  int res = rte_mempool_get(p->fdMp, (void**)&entry);
  if (unlikely(res != 0)) {
    N_LOGE("New fd-alloc-err" N_LOG_ERROR_BLANK);
    return NULL;
  }

  res = FileServerFd_InvokeStatx(p, entry, dfd, filename, now);
  if (unlikely(res != 0)) {
    N_LOGD("New statx-err mount=%d dfd=%d filename=%s" N_LOG_ERROR_ERRNO, mount, dfd, filename,
           res);
    goto FAIL;
  }

  const char* logFilename = NULL;
  if (likely(filename[0] != '\0')) {
    logFilename = filename;
    entry->fd = syscall(SYS_openat2, dfd, filename, &p->openHow, sizeof(p->openHow));
  } else {
    logFilename = "(empty)";
    entry->fd = dup(dfd);
  }

  if (unlikely(entry->fd < 0)) {
    N_LOGD("New openat2-err mount=%d dfd=%d filename=%s" N_LOG_ERROR_ERRNO, mount, dfd, logFilename,
           errno);
    goto FAIL;
  }
  ++p->cnt.fdNew;

  entry->refcnt = 1;
  entry->lsL = UINT32_MAX;
  entry->prefixL = prefix.length;
  rte_memcpy(entry->nameV, prefix.value, prefix.length);
  FileServerFd_PrepareVersionedName(p, entry);
  FileServerFd_PrepareMetaInfo(p, entry, entry->st.stx_size);

  HASH_ADD_BYHASHVALUE(hh, p->fdHt, self, 0, hash, entry);
  N_LOGD("New mount=%d dfd=%d filename=%s fd=%d version=%" PRIu64 " size=%" PRIu64, mount, dfd,
         logFilename, entry->fd, entry->version, (uint64_t)entry->st.stx_size);
  return entry;

FAIL:
  rte_mempool_put(p->fdMp, entry);
  ++p->cnt.fdNotFound;
  return FileServer_NotFound;
}

FileServerFd*
FileServerFd_Open(FileServer* p, const PName* name, TscTime now) {
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
FileServerFd_Unref(FileServer* p, FileServerFd* entry) {
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
FileServerFd_Clear(FileServer* p) {
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

uint32_t
FileServerFd_PrepareMetadata_(FileServer* p, FileServerFd* entry) {
  uint8_t* output = entry->metadataV;

#define HAS_STAT_BIT(bit)                                                                          \
  (likely((FileServerStatxRequired & (bit)) == (bit) || FileServerFd_HasStatBit(entry, (bit))))

#define APPEND_NNI(type, bits, val)                                                                \
  do {                                                                                             \
    struct {                                                                                       \
      unaligned_uint32_t tl;                                                                       \
      unaligned_uint##bits##_t v;                                                                  \
    } __rte_packed* f = (void*)output;                                                             \
    f->tl = TlvEncoder_ConstTL3(TtFile##type, sizeof(f->v));                                       \
    f->v = rte_cpu_to_be_##bits((uint##bits##_t)(val));                                            \
    output += sizeof(*f);                                                                          \
  } while (false)

  if (likely(FileServerFd_IsFile(entry))) {
    NDNDPDK_ASSERT(entry->meta[2] == TtFinalBlock);
    rte_memcpy(output, &entry->meta[2], entry->meta[1]);
    output += entry->meta[1];
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
  entry->metadataL = RTE_PTR_DIFF(output, entry->metadataV);
  NDNDPDK_ASSERT(entry->metadataL <= RTE_DIM(entry->metadataV));
  return FileServerFd_SizeofMetadata_(entry);
}

void
FileServerFd_WriteMetadata(FileServerFd* entry, struct iovec* iov, int iovcnt) {
  uint8_t nameTL[L3TypeLengthHeadroom] = {TtName};
  uint16_t sizeofNameTL = 1 + TlvEncoder_WriteVarNum(&nameTL[1], entry->versionedL);
  uint32_t nCopied = 0;

  struct spdk_iov_xfer ix;
  spdk_iov_xfer_init(&ix, iov, iovcnt);
  nCopied += spdk_iov_xfer_from_buf(&ix, nameTL, sizeofNameTL);
  nCopied += spdk_iov_xfer_from_buf(&ix, entry->nameV, entry->versionedL);
  nCopied += spdk_iov_xfer_from_buf(&ix, entry->metadataV, entry->metadataL);

  NDNDPDK_ASSERT(nCopied == FileServerFd_SizeofMetadata_(entry));

  struct iovec remIov[1];
  int remIovcnt = RTE_DIM(remIov);
  Mbuf_RemainingIovec(ix, remIov, &remIovcnt);
  NDNDPDK_ASSERT(remIovcnt == 0);
}

__attribute__((nonnull)) static inline int
FileServerFd_DirentType(FileServerFd* entry, struct dirent64* de) {
  switch (de->d_type) {
    case DT_UNKNOWN:
    case DT_LNK:
      break;
    default:
      return de->d_type;
  }

  struct statx st;
  int res = statx(entry->fd, de->d_name, 0, STATX_TYPE, &st);
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
FileServerFd_GenerateLs(FileServer* p, FileServerFd* entry) {
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
  FileServerFd_PrepareMetaInfo(p, entry, entry->lsL);
  N_LOGD("Ls generated fd=%d version=%" PRIu64 " length=%" PRIu32 " full=%d", entry->fd,
         entry->version, entry->lsL, (int)isFull);
  return true;

FAIL:
  entry->lsL = UINT32_MAX;
  return false;
}
