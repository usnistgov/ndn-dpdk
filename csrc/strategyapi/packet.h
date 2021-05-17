#ifndef NDNDPDK_STRATEGYAPI_PACKET_H
#define NDNDPDK_STRATEGYAPI_PACKET_H

/** @file */

#include "common.h"

typedef struct SgPacket
{
  char a_[22];
  FaceID rxFace;
  char b_[8];
  union
  {
    uint32_t packet_type_;
    struct
    {
      uint16_t c_ : 16;
      uint8_t l2type : 4;
      uint8_t l3type : 4;
    };
  };
  char d_[92];
  char mbuf_end_[0];
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

#endif // NDNDPDK_STRATEGYAPI_PACKET_H
