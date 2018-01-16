#ifndef NDN_DPDK_CONTAINER_NAMESET_NAMESET_H
#define NDN_DPDK_CONTAINER_NAMESET_NAMESET_H

/// \file

#include "../../ndn/name.h"

typedef struct NameSetRecord NameSetRecord;

/** \brief An unordered set of names.
 *
 *  This data structure has sequential lookups and is only suitable for small sets.
 */
typedef struct NameSet
{
  NameSetRecord** records;
  int nRecords;
} NameSet;

/** \brief Release all memory allocated by NameSet.
 */
void NameSet_Close(NameSet* set);

/** \brief Insert a name.
 *  \param comps encoding of name components.
 *  \param compsLen length of \p comps.
 *  \param usr extra user information, NULL to initialize as zeros.
 *  \param usrLen length of extra user information.
 *  \warning Crash if memory allocation fails.
 */
void NameSet_Insert(NameSet* set, const uint8_t* comps, uint16_t compsLen,
                    const void* usr, size_t usrLen);

/** \brief Erase a name at \p index .
 */
void NameSet_Erase(NameSet* set, int index);

/** \brief Get the name at \p index .
 *  \param[out] compsLen length of returned comps.
 *  \return encoding of name components.
 */
const uint8_t* NameSet_GetName(const NameSet* set, int index,
                               uint16_t* compsLen);

/** \brief Get extra user information at \p index .
 */
void* NameSet_GetUsr(const NameSet* set, int index);

/** \brief Get extra user information as \p index and cast to type T.
 */
#define NameSet_GetUsrT(set, index, T) (T)(NameSet_GetUsr((set), (index)))

/** \brief Determine if a name exists.
 *  \return index within NameSet, or -1 if not found.
 */
int NameSet_FindExact(const NameSet* set, const uint8_t* comps,
                      uint16_t compsLen);

/** \brief Determine if any name in the set is a prefix of queried name.
 *  \return index within NameSet, or -1 if not found.
 */
int NameSet_FindPrefix(const NameSet* set, const uint8_t* comps,
                       uint16_t compsLen);

#endif // NDN_DPDK_CONTAINER_NAMESET_NAMESET_H