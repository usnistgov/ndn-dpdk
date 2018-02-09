#include "data-pkt.h"
#include "tlv-encoder.h"

NdnError
DecodeData(TlvDecodePos* d, DataPkt* data)
{
  TlvElement dataEle;
  NdnError e = DecodeTlvElementExpectType(d, TT_Data, &dataEle);
  RETURN_IF_UNLIKELY_ERROR;

  memset(data, 0, sizeof(DataPkt));

  TlvDecodePos d1;
  TlvElement_MakeValueDecoder(&dataEle, &d1);

  e = DecodeName(&d1, &data->name);
  RETURN_IF_UNLIKELY_ERROR;

  {
    TlvElement metaEle;
    e = DecodeTlvElementExpectType(&d1, TT_MetaInfo, &metaEle);
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

void
EncodeData1(struct rte_mbuf* m, const Name* name, struct rte_mbuf* payload)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeData1_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >= EncodeData1_GetTailroom(name));

  TlvEncoder* en = MakeTlvEncoder(m);

  MbufLoc mlName;
  MbufLoc_Copy(&mlName, &name->comps[0].pos);
  MbufLoc_ReadTo(&mlName, rte_pktmbuf_append(m, name->nOctets), name->nOctets);
  PrependVarNum(en, name->nOctets);
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
