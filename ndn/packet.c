#include "packet.h"

NdnError
Packet_ParseL3(Packet* npkt)
{
  TlvDecodePos d;
  MbufLoc_Init(&d, Packet_ToMbuf(npkt));
  TlvElement ele;
  NdnError e = DecodeTlvElement(&d, &ele);
  RETURN_IF_UNLIKELY_ERROR;

  switch (ele.type) {
    case TT_Interest:
      Packet_SetL3PktType(npkt, L3PktType_Interest);
      assert(false); // not implemented
      return NdnError_BadType;
    case TT_Data:
      Packet_SetL3PktType(npkt, L3PktType_Data);
      return PData_FromElement(Packet_GetDataHdr(npkt), &ele);
  }
  return NdnError_BadType;
}
