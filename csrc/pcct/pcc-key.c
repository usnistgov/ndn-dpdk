#include "pcc-key.h"
#include "debug-string.h"
#include "pcc-entry.h"

static_assert(sizeof(PccKeyExt) <= sizeof(PccEntry), "");

const char*
PccSearch_ToDebugString(const PccSearch* search)
{
  return "";
  // PccDebugString_Clear();

  // char nameStr[LNAME_MAX_STRING_SIZE + 1];
  // if (LName_ToString(search->name, nameStr, sizeof(nameStr)) == 0) {
  //   snprintf(nameStr, sizeof(nameStr), "(empty)");
  // }
  // PccDebugString_Appendf("name=%s", nameStr);

  // if (LName_ToString(search->fh, nameStr, sizeof(nameStr)) == 0) {
  //   snprintf(nameStr, sizeof(nameStr), "(empty)");
  // }
  // return PccDebugString_Appendf(" fh=%s", nameStr);
}
