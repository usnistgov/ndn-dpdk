#include "encode-data.h"
#include "tlv-encoder.h"

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

const uint16_t __EncodeData_FakeSigLen = sizeof(FAKESIG);

void
__EncodeData(struct rte_mbuf* m, uint16_t nameL, const uint8_t* nameV,
             uint32_t freshnessPeriod, uint16_t contentL,
             const uint8_t* contentV)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeData_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >= EncodeData_GetTailroom(nameL, contentL));
  TlvEncoder* en = MakeTlvEncoder(m);

  {
    AppendVarNum(en, TT_Name);
    AppendVarNum(en, nameL);
    rte_memcpy(rte_pktmbuf_append(m, nameL), nameV, nameL);
  }

  if (freshnessPeriod != 0) {
    typedef struct MetaInfoF
    {
      uint8_t metaInfoT;
      uint8_t metaInfoL;
      uint8_t freshnessPeriodT;
      uint8_t freshnessPeriodL;
      rte_be32_t freshnessPeriodV;
    } __rte_packed MetaInfoF;

    MetaInfoF* f = (MetaInfoF*)TlvEncoder_Append(en, sizeof(MetaInfoF));
    f->metaInfoT = TT_MetaInfo;
    f->metaInfoL = 6;
    f->freshnessPeriodT = TT_FreshnessPeriod;
    f->freshnessPeriodL = 4;
    *(unaligned_uint32_t*)&f->freshnessPeriodV =
      rte_cpu_to_be_32(freshnessPeriod);
  }

  if (contentL != 0) {
    AppendVarNum(en, TT_Content);
    AppendVarNum(en, contentL);
    rte_memcpy(rte_pktmbuf_append(m, contentL), contentV, contentL);
  }

  rte_memcpy(rte_pktmbuf_append(m, __EncodeData_FakeSigLen), FAKESIG,
             __EncodeData_FakeSigLen);

  PrependVarNum(en, m->pkt_len);
  PrependVarNum(en, TT_Data);
}
