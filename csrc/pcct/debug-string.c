#include "debug-string.h"

// typedef struct PccDebugString
// {
//   int len;
//   char s[PccDebugStringLength];
// } PccDebugString;
// RTE_DEFINE_PER_LCORE(PccDebugString, gPccDebugString);

void
PccDebugString_Clear()
{
  // RTE_PER_LCORE(gPccDebugString).len = 0;
  // RTE_PER_LCORE(gPccDebugString).s[0] = '\0';
}

const char*
PccDebugString_Appendf(const char* fmt, ...)
{
  return "";
  // char* begin = RTE_PER_LCORE(gPccDebugString).s;
  // int* len = &RTE_PER_LCORE(gPccDebugString).len;
  // char* output = RTE_PTR_ADD(begin, *len);
  // int room = PccDebugStringLength - *len;

  // va_list args;
  // va_start(args, fmt);
  // int res = vsnprintf(output, room, fmt, args);
  // va_end(args);

  // if (res < 0) {
  //   *output = '\0';
  // } else if (res >= room) {
  //   *len += room - 1;
  // } else {
  //   *len += res;
  // }

  // return begin;
}
