#include "naming.h"

FileServerRequestName
FileServer_ParseRequest(const PInterest* pi)
{
  FileServerRequestName rn = { 0 };
  if (unlikely(pi->name.firstNonGeneric < 0)) {
    goto FAIL;
  }
  LName suffix = PName_Slice(&pi->name, pi->name.firstNonGeneric, INT16_MAX);

  uint16_t pos = 0, type = 0, length = 0;
  while (likely(LName_Component(suffix, &pos, &type, &length))) {
    const uint8_t* value = &suffix.value[pos];
    pos += length;
    switch (type) {
      case TtVersionNameComponent:
        if (likely(Nni_Decode(length, value, &rn.version))) {
          rn.kind = rn.kind | FileServerRequestVersion;
        } else {
          goto FAIL;
        }
        break;
      case TtSegmentNameComponent:
        if (likely(Nni_Decode(length, value, &rn.segment))) {
          rn.kind = rn.kind | FileServerRequestSegment;
        } else {
          goto FAIL;
        }
        break;
      case TtKeywordNameComponent:
        switch (length) {
          case sizeof(FileServer_KeywordLs) - 2:
            if (likely(memcmp(value, &FileServer_KeywordLs[2], sizeof(FileServer_KeywordLs) - 2) ==
                       0)) {
              rn.kind = rn.kind | FileServerRequestLs;
            } else {
              goto FAIL;
            }
            break;
          case sizeof(FileServer_KeywordMetadata) - 2:
            if (likely(memcmp(value, &FileServer_KeywordMetadata[2],
                              sizeof(FileServer_KeywordMetadata) - 2) == 0 &&
                       pi->canBePrefix && pi->mustBeFresh)) {
              rn.kind = rn.kind | FileServerRequestMetadata;
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
  return rn;

FAIL:
  rn.kind = FileServerRequestNone;
  return rn;
}

bool
FileServer_ToFilename(const PName* name, int16_t mountComps, char filename[PATH_MAX])
{
  LName path = PName_Slice(name, mountComps, name->firstNonGeneric);
  if (unlikely(path.length >= PATH_MAX)) {
    return false;
  }

  char* output = &filename[0];
  uint16_t pos = 0, type = 0, length = 0;
  while (likely(LName_Component(path, &pos, &type, &length))) {
    if (unlikely(length == 0 || length > NAME_MAX)) {
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
    if (unlikely(allPeriods && length <= 2)) {
      return false;
    }
  }
  *output++ = '\0';
  return true;
}
