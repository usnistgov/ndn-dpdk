#include "pit-entry.h"
#include "debug-string.h"
#include "pit.h"

const char*
PitEntry_ToDebugString(PitEntry* entry)
{
  PccDebugString_Clear();

  PInterest* interest = Packet_GetInterestHdr(entry->npkt);
  char nameStr[LNAME_MAX_STRING_SIZE + 1];
  if (LName_ToString(*(LName*)&interest->name, nameStr, sizeof(nameStr)) == 0) {
    snprintf(nameStr, sizeof(nameStr), "(empty)");
  }

  PccDebugString_Appendf("%s CBP=%" PRIu8 " MBF=%d DN=[", nameStr,
                         entry->nCanBePrefix, (int)entry->mustBeFresh);
  for (int index = 0; index < PIT_ENTRY_MAX_DNS; ++index) {
    PitDn* dn = &entry->dns[index];
    if (dn->face == FACEID_INVALID) {
      break;
    }
    PccDebugString_Appendf("%" PRI_FaceId ",", dn->face);
  }
  PccDebugString_Appendf("] UP=[");
  for (int index = 0; index < PIT_ENTRY_MAX_UPS; ++index) {
    PitUp* up = &entry->ups[index];
    if (up->face == FACEID_INVALID) {
      break;
    }
    PccDebugString_Appendf("%" PRI_FaceId ",", up->face);
  }
  return PccDebugString_Appendf("]");
}
