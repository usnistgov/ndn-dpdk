#include "data.h"

NdnError
PData_FromPacket(PData* data, struct rte_mbuf* pkt, struct rte_mempool* nameMp)
{
  TlvDecodePos d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement dataEle;
  NdnError e = DecodeTlvElementExpectType(&d0, TT_Data, &dataEle);
  RETURN_IF_UNLIKELY_ERROR;

  TlvDecodePos d1;
  TlvElement_MakeValueDecoder(&dataEle, &d1);

  TlvElement nameEle;
  e = DecodeTlvElementExpectType(&d1, TT_Name, &nameEle);
  RETURN_IF_UNLIKELY_ERROR;
  data->name.v = TlvElement_LinearizeValue(&nameEle, pkt, nameMp, &d1);
  RETURN_IF_UNLIKELY_NULL(data->name.v, NdnError_AllocError);
  e = PName_Parse(&data->name.p, nameEle.length, data->name.v);
  RETURN_IF_UNLIKELY_ERROR;

  data->freshnessPeriod = 0;
  TlvElement metaEle;
  e = DecodeTlvElementExpectType(&d1, TT_MetaInfo, &metaEle);
  if (e == NdnError_Incomplete || e == NdnError_BadType) {
    return NdnError_OK; // MetaInfo not present
  }
  RETURN_IF_UNLIKELY_ERROR;

  TlvDecodePos d2;
  TlvElement_MakeValueDecoder(&metaEle, &d2);
  while (!MbufLoc_IsEnd(&d2)) {
    TlvElement metaChild;
    e = DecodeTlvElement(&d2, &metaChild);
    RETURN_IF_UNLIKELY_ERROR;

    if (metaChild.type != TT_FreshnessPeriod) {
      continue; // ignore other children of MetaInfo
    }

    uint64_t fpV;
    bool ok = TlvElement_ReadNonNegativeInteger(&metaChild, &fpV);
    RETURN_IF_UNLIKELY_ERROR;
    if (unlikely(fpV > UINT32_MAX)) {
      data->freshnessPeriod = UINT32_MAX;
    } else {
      data->freshnessPeriod = (uint32_t)fpV;
    }
    break;
  }

  return NdnError_OK;
}
