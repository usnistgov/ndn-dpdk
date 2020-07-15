#ifndef NDNDPDK_MINTMRTEST_H
#define NDNDPDK_MINTMRTEST_H

#include "../../../csrc/mintmr/mintmr.h"

typedef struct MinTmrTestRecord
{
  MinTmr tmr;
  int triggered;
} MinTmrTestRecord;

MinTmrTestRecord records[6];

#endif // NDNDPDK_MINTMRTEST_H
