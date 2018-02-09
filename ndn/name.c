#include "name.h"

uint64_t
LName_ComputeHash(LName n)
{
  SipHash h;
  SipHash_Init(&h, &theNameHashKey);
  SipHash_Write(&h, n.value, n.length);
  return SipHash_Final(&h);
}

NameCompareResult
LName_Compare(LName lhs, LName rhs)
{
  uint16_t minOctets = lhs.length <= rhs.length ? lhs.length : rhs.length;
  int cmp = memcmp(lhs.value, rhs.value, minOctets);
  if (cmp != 0) {
    return ((cmp > 0) - (cmp < 0)) << 1;
  }
  cmp = lhs.length - rhs.length;
  return (cmp > 0) - (cmp < 0);
}
