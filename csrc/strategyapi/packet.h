#ifndef NDN_DPDK_STRATEGYAPI_PACKET_H
#define NDN_DPDK_STRATEGYAPI_PACKET_H

/** @file */

#include "common.h"

typedef struct SgPacket
{
  char _a[22];
  FaceID rxFace;
  char _b[8];
  union
  {
    uint32_t _packet_type;
    struct
    {
      uint16_t _c : 16;
      uint8_t l2type : 4;
      uint8_t l3type : 4;
    };
  };
  char _d[20];
  TscTime timestamp;
  char _e[64];
  char _mbuf_end[0];
  char _f[8];
  uint8_t nackReason;
  uint8_t congMark;
} SgPacket;

typedef enum SgNackReason
{
  SgNackCongestion = 50,
  SgNackDuplicate = 100,
  SgNackNoRoute = 150,
  SgNackUnspecified = 255,
} SgNackReason;

#endif // NDN_DPDK_STRATEGYAPI_PACKET_H
