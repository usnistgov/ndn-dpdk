#ifndef NDNDPDK_FILESERVER_NAMING_H
#define NDNDPDK_FILESERVER_NAMING_H

/** @file */

#include "../ndni/interest.h"
#include "../ndni/nni.h"
#include "enum.h"

/** @brief 32=ls keyword component. */
static const uint8_t FileServer_KeywordLs[4] = { TtKeywordNameComponent, 2, 0x6C, 0x73 };

/** @brief 32=metadata keyword component. */
static const uint8_t FileServer_KeywordMetadata[10] = {
  TtKeywordNameComponent, 8, 0x6D, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61
};

enum
{
  /**
   * @brief Maximum mount+path TLV-LENGTH to accommodate [32=ls]+[32=metadata]+version+segment
   *        suffix components.
   */
  FileServer_MaxPrefixL =
    NameMaxLength - sizeof(FileServer_KeywordLs) - sizeof(FileServer_KeywordMetadata) - 10 - 10,
};

/** @brief Indicate what components are present in Interest name. */
typedef enum FileServerRequestKind
{
  FileServerRequestNone = 0,
  FileServerRequestVersion = 1 << 0,
  FileServerRequestSegment = 1 << 1,
  FileServerRequestLs = 1 << 2,
  FileServerRequestMetadata = 1 << 3,
} FileServerRequestKind;

/** @brief Parsed Interest name processed by file server. */
typedef struct FileServerRequestName
{
  uint64_t version;
  uint64_t segment;
  FileServerRequestKind kind;
} FileServerRequestName;

/** @brief Get mount + path prefix. */
__attribute__((nonnull)) static inline LName
FileServer_GetPrefix(const PName* name)
{
  return PName_GetPrefix(name, name->firstNonGeneric);
}

/** @brief Parse Interest name. */
__attribute__((nonnull)) FileServerRequestName
FileServer_ParseRequest(const PInterest* pi);

/**
 * @brief Construct relative filename.
 * @param mountComps number of components in mount prefix.
 * @param[out] filename relative filename.
 */
__attribute__((nonnull)) bool
FileServer_ToFilename(const PName* name, int16_t mountComps, char filename[PATH_MAX]);

#endif // NDNDPDK_FILESERVER_NAMING_H
