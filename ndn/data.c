#include "data.h"
#include "tlv-encoder.h"

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
