#ifndef NDNDPDK_FILESERVER_NAMING_H
#define NDNDPDK_FILESERVER_NAMING_H

/** @file */

#include "../ndni/name.h"
#include "../ndni/nni.h"
#include "enum.h"

__attribute__((nonnull)) static inline uint16_t
FileServer_NameToPath(LName name, uint16_t prefixL, char* output, size_t capacity)
{
  if (unlikely((size_t)(name.length - prefixL) >= capacity)) {
    return UINT16_MAX;
  }
  uint16_t off = prefixL;
  size_t pos = 0;
  while (off + 2 < name.length) {
    uint8_t typ = name.value[off];
    if (unlikely(typ != TtGenericNameComponent)) {
      break;
    }
    uint8_t len = name.value[off + 1];
    if (unlikely(len >= RTE_MIN(0xFD, NAME_MAX))) {
      return UINT16_MAX;
    }
    off += 2;

    bool allPeriods = true;
    for (uint8_t i = 0; i < len; ++i) {
      char ch = (char)name.value[off++];
      switch (ch) {
        case '\0':
        case '/':
          return UINT16_MAX;
        case '.':
          break;
        default:
          allPeriods = false;
          break;
      }
      output[pos++] = ch;
    }
    if (unlikely(len <= 2 && allPeriods)) {
      return UINT16_MAX;
    }
  }
  output[pos++] = '\0';
  return off;
}

typedef struct FileServerSuffix
{
  uint64_t version;
  uint64_t segment;
  bool ok;
  bool hasVersion;
  bool hasSegment;
  bool isLs;
  bool isMetadata;
} FileServerSuffix;

__attribute__((nonnull)) static inline FileServerSuffix
FileServer_ParseSuffix(LName name, uint16_t prefixL)
{
  FileServerSuffix result = { 0 };
  uint16_t off = prefixL;
  while (off + 2 < name.length) {
    uint8_t typ = name.value[off];
    uint8_t len = name.value[off + 1];
    if (unlikely(len >= 0xFD)) {
      goto FAIL;
    }
    off += 2;

    switch (typ) {
      case TtVersionNameComponent:
        result.hasVersion = Nni_Decode(len, &name.value[off], &result.version);
        if (unlikely(!result.hasVersion)) {
          goto FAIL;
        }
        break;
      case TtSegmentNameComponent:
        result.hasSegment = Nni_Decode(len, &name.value[off], &result.segment);
        if (unlikely(!result.hasSegment)) {
          goto FAIL;
        }
        break;
      case TtKeywordNameComponent:
        switch (len) {
          case 2:
            if (likely(memcmp(&name.value[off], "ls", 2) == 0)) {
              result.isLs = true;
            } else {
              goto FAIL;
            }
            break;
          case 8:
            if (likely(memcmp(&name.value[off], "metadata", 8) == 0)) {
              result.isMetadata = true;
            } else {
              goto FAIL;
            }
            break;
          default:
            goto FAIL;
        }
        break;
      default:
        goto FAIL;
    }
  }

  result.ok = true;
FAIL:
  return result;
}

#endif // NDNDPDK_FILESERVER_NAMING_H
