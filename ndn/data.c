#include "data.h"
#include "tlv-encoder.h"

NdnError
PData_FromElement(PData* data, const TlvElement* ele)
{
  assert(ele->type == TT_Data);

  TlvDecodePos d;
  TlvElement_MakeValueDecoder(ele, &d);

  {
    TlvElement nameEle;
    NdnError e = DecodeTlvElementExpectType(&d, TT_Name, &nameEle);
    RETURN_IF_UNLIKELY_ERROR;
    e = PName_FromElement(&data->name.p, &nameEle);
    RETURN_IF_UNLIKELY_ERROR;
    data->name.v = TlvElement_GetLinearValue(&nameEle);
  }

  {
    data->freshnessPeriod = 0;
    TlvElement metaEle;
    NdnError e = DecodeTlvElementExpectType(&d, TT_MetaInfo, &metaEle);
    if (e == NdnError_Incomplete || e == NdnError_BadType) {
      return NdnError_OK; // MetaInfo not present
    }
    RETURN_IF_UNLIKELY_ERROR;

    TlvDecodePos d1;
    TlvElement_MakeValueDecoder(&metaEle, &d1);
    while (!MbufLoc_IsEnd(&d1)) {
      TlvElement metaChild;
      e = DecodeTlvElement(&d1, &metaChild);
      RETURN_IF_UNLIKELY_ERROR;

      if (metaChild.type != TT_FreshnessPeriod) {
        continue; // ignore other children of MetaInfo
      }

      uint64_t fpVal;
      bool ok = TlvElement_ReadNonNegativeInteger(&metaChild, &fpVal);
      RETURN_IF_UNLIKELY_ERROR;
      data->freshnessPeriod =
        unlikely(fpVal > UINT32_MAX) ? UINT32_MAX : (uint32_t)fpVal;
      break;
    }
  }

  return NdnError_OK;
}

void
EncodeData1(struct rte_mbuf* m, LName name, struct rte_mbuf* payload)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeData1_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >= EncodeData1_GetTailroom(name.length));

  TlvEncoder* en = MakeTlvEncoder(m);

  rte_memcpy(rte_pktmbuf_append(m, name.length), name.value, name.length);
  PrependVarNum(en, name.length);
  PrependVarNum(en, TT_Name);

  AppendVarNum(en, TT_MetaInfo);
  AppendVarNum(en, 0);

  AppendVarNum(en, TT_Content);
  AppendVarNum(en, payload->pkt_len);

  rte_pktmbuf_chain(m, payload);
}

// clang-format off
static const uint8_t FAKESIG[] = {
  TT_SignatureInfo, 0x03,
    TT_SignatureType, 0x01, 0x00,
  TT_SignatureValue, 0x20,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
};
// clang-format on

const uint16_t __EncodeData2_FakeSigLen = sizeof(FAKESIG);

void
EncodeData2(struct rte_mbuf* m, struct rte_mbuf* data1)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeData2_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >= EncodeData2_GetTailroom());
  MakeTlvEncoder(m); // asserts empty

  char* room = rte_pktmbuf_append(m, __EncodeData2_FakeSigLen);
  rte_memcpy(room, FAKESIG, __EncodeData2_FakeSigLen);

  rte_pktmbuf_chain(data1, m);
}

void
EncodeData3(struct rte_mbuf* data2)
{
  TlvEncoder* en = MakeTlvEncoder_Unchecked(data2);
  PrependVarNum(en, data2->pkt_len);
  PrependVarNum(en, TT_Data);
}
