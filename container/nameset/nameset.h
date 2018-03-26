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
  int numaSocket; ///< where to allocate memory for new records
} NameSet;

/** \brief Release all memory allocated by NameSet.
 */
void NameSet_Close(NameSet* set);

void __NameSet_Insert(NameSet* set, uint16_t nameL, const uint8_t* nameV,
                      const void* usr, size_t usrLen);

/** \brief Insert a name.
 *  \param usr extra user information, NULL to initialize as zeros.
 *  \param usrLen length of extra user information.
 *  \warning Crash if memory allocation fails.
 */
static void
NameSet_Insert(NameSet* set, LName name, const void* usr, size_t usrLen)
{
  __NameSet_Insert(set, name.length, name.value, usr, usrLen);
}

/** \brief Erase a name at \p index.
 */
void NameSet_Erase(NameSet* set, int index);

/** \brief Get the name at \p index.
 */
LName NameSet_GetName(const NameSet* set, int index);

/** \brief Get extra user information at \p index.
 */
void* NameSet_GetUsr(const NameSet* set, int index);

/** \brief Get extra user information as \p index and cast to type T.
 */
#define NameSet_GetUsrT(set, index, T) (T)(NameSet_GetUsr((set), (index)))

int __NameSet_FindExact(const NameSet* set, uint16_t nameL,
                        const uint8_t* nameV);

/** \brief Determine if a name exists.
 *  \return index within NameSet, or -1 if not found.
 */
static int
NameSet_FindExact(const NameSet* set, LName name)
{
  return __NameSet_FindExact(set, name.length, name.value);
}

int __NameSet_FindPrefix(const NameSet* set, uint16_t nameL,
                         const uint8_t* nameV);

/** \brief Determine if any name in the set is a prefix of queried name.
 *  \return index within NameSet, or -1 if not found.
 */
static int
NameSet_FindPrefix(const NameSet* set, LName name)
{
  return __NameSet_FindPrefix(set, name.length, name.value);
}

#endif // NDN_DPDK_CONTAINER_NAMESET_NAMESET_H