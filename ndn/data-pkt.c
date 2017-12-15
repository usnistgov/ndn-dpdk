#include "data-pkt.h"

NdnError
DecodeData(TlvDecoder* d, DataPkt* data)
{
  TlvElement dataEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_Data, &dataEle);
  RETURN_IF_UNLIKELY_ERROR;

  memset(data, 0, sizeof(DataPkt));

  TlvDecoder d1;
  TlvElement_MakeValueDecoder(&dataEle, &d1);

  e = DecodeName(&d1, &data->name);
  RETURN_IF_UNLIKELY_ERROR;

  {
    TlvElement metaEle;
    e = DecodeTlvElementExpectType(&d1, TT_MetaInfo, &metaEle);
    RETURN_IF_UNLIKELY_ERROR;

    TlvDecoder d2;
    TlvElement_MakeValueDecoder(&metaEle, &d2);
    while (!MbufLoc_IsEnd(&d2)) {
      TlvElement metaChild;
      e = DecodeTlvElement(&d2, &metaChild);
      RETURN_IF_UNLIKELY_ERROR;

      if (metaChild.type != TT_FreshnessPeriod) {
        continue; // ignore other children of MetaInfo
      }

      uint64_t fpVal;
      bool ok = TlvElement_ReadNonNegativeInteger(&metaChild, &fpVal);
      if (unlikely(!ok) || fpVal >= UINT32_MAX) {
        return NdnError_BadFreshnessPeriod;
      }
      data->freshnessPeriod = (uint32_t)fpVal;
      break;
    }
  }

  {
    TlvElement contentEle;
    e = DecodeTlvElementExpectType(&d1, TT_Content, &contentEle);
    RETURN_IF_UNLIKELY_ERROR;
    TlvElement_MakeValueDecoder(&contentEle, &data->content);
  }

  // ignore Signature

  return NdnError_OK;
}