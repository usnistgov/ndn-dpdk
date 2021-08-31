#ifndef NDNDPDK_FILESERVER_NAMING_H
#define NDNDPDK_FILESERVER_NAMING_H

/** @file */

#include "../ndni/name.h"
#include "../ndni/nni.h"
#include "enum.h"

/** @brief Parsed Interest name processed by file server. */
typedef struct FileServerRequestName
{
  uint64_t version; ///< version number
  uint64_t segment; ///< segment number
  bool hasVersion;  ///< version number exists
  bool hasSegment;  ///< segment number exists
  bool isLs;        ///< is directory listing request
  bool isMetadata;  ///< is metadata request
} FileServerRequestName;

/**
 * @brief Parse Interest name.
 * @param[out] p parse result.
 * @param name Interest name.
 * @return whether success.
 */
static inline bool
FileServer_ParseRequest(FileServerRequestName* p, const PName* name)
{
  if (unlikely(name->firstNonGeneric < 0)) {
    return false;
  }
  LName suffix = PName_Slice(name, name->firstNonGeneric, INT16_MAX);
  *p = (FileServerRequestName){ 0 };

  uint16_t pos = 0, type = 0, length = 0;
  while (likely(LName_Component(suffix, &pos, &type, &length))) {
    const uint8_t* value = &suffix.value[pos];
    pos += length;
    switch (type) {
      case TtVersionNameComponent:
        p->hasVersion = Nni_Decode(length, value, &p->version);
        if (unlikely(!p->hasVersion)) {
          return false;
        }
        break;
      case TtSegmentNameComponent:
        p->hasSegment = Nni_Decode(length, value, &p->segment);
        if (unlikely(!p->hasSegment)) {
          return false;
        }
        break;
      case TtKeywordNameComponent:
        switch (length) {
          case 2:
            if (likely(memcmp(value, "ls", 2) == 0)) {
              p->isLs = true;
            } else {
              return false;
            }
            break;
          case 8:
            if (likely(memcmp(value, "metadata", 8) == 0)) {
              p->isMetadata = true;
            } else {
              return false;
            }
            break;
          default:
            return false;
        }
        break;
      default:
        return false;
    }
  }

  return true;
}

/**
 * @brief Construct relative filename.
 * @param mountComps number of components in mount prefix.
 * @param[out] filename relative filename.
 */
static inline bool
FileServer_ToFilename(const PName* name, int16_t mountComps, char filename[PATH_MAX])
{
  LName path = PName_Slice(name, mountComps, name->firstNonGeneric);
  if (unlikely(path.length >= PATH_MAX)) {
    return false;
  }

  char* output = &filename[0];
  uint16_t pos = 0, type = 0, length = 0;
  while (likely(LName_Component(path, &pos, &type, &length))) {
    if (unlikely(length > NAME_MAX)) {
      return false;
    }
    const uint8_t* value = &path.value[pos];
    pos += length;
    const uint8_t* valueEnd = &path.value[pos];

    if (output != filename) {
      *output++ = '/';
    }

    bool allPeriods = false;
    while (value != valueEnd) {
      char ch = (char)*value++;
      switch (ch) {
        case '\0':
        case '/':
          return false;
        case '.':
          break;
        default:
          allPeriods = false;
          break;
      }
      *output++ = ch;
    }
    if (unlikely(allPeriods)) {
      return false;
    }
  }
  *output++ = '\0';
  return true;
}

#endif // NDNDPDK_FILESERVER_NAMING_H
