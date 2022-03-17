#ifndef NDNDPDK_STRATEGYAPI_PACKET_H
#define NDNDPDK_STRATEGYAPI_PACKET_H

/** @file */

#include "../ndni/an.h"
#include "common.h"

typedef struct SgPacket
{
  uint8_t a_[22];
  FaceID rxFace;
  uint8_t b_[8];
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
  uint8_t d_[92];
  RTE_MARKER mbuf_end_;
  uint8_t nackReason;
  uint8_t congMark;
} SgPacket;

#endif // NDNDPDK_STRATEGYAPI_PACKET_H
