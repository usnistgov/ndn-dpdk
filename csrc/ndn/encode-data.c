#include "encode-data.h"
#include "tlv-encoder.h"

// clang-format off
static const uint8_t FAKESIG[] = {
  TtSignatureInfo, 0x03,
    TtSignatureType, 0x01, 0x00,
  TtSignatureValue, 0x20,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
};
// clang-format on

const uint16_t EncodeData_FakeSigLen_ = sizeof(FAKESIG);

static void
EncodeData_AppendNameNoSuffix(TlvEncoder* en,
                              uint16_t namePrefixL,
                              const uint8_t* namePrefixV,
                              uint16_t nameSuffixL)
{
  struct rte_mbuf* m = TlvEncoder_AsMbuf(en);
  AppendVarNum(en, TtName);
  AppendVarNum(en, namePrefixL + nameSuffixL);
  if (likely(namePrefixL > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, namePrefixL), namePrefixV, namePrefixL);
  }
}

static void
EncodeData_AppendFreshnessContentSignature(TlvEncoder* en,
                                           uint32_t freshnessPeriod,
                                           uint16_t contentL,
                                           const uint8_t* contentV)
{
  struct rte_mbuf* m = TlvEncoder_AsMbuf(en);

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
    f->metaInfoT = TtMetaInfo;
    f->metaInfoL = 6;
    f->freshnessPeriodT = TtFreshnessPeriod;
    f->freshnessPeriodL = 4;
    *(unaligned_uint32_t*)&f->freshnessPeriodV =
      rte_cpu_to_be_32(freshnessPeriod);
  }

  if (contentL != 0) {
    AppendVarNum(en, TtContent);
    AppendVarNum(en, contentL);
    rte_memcpy(rte_pktmbuf_append(m, contentL), contentV, contentL);
  }

  rte_memcpy(rte_pktmbuf_append(m, EncodeData_FakeSigLen_),
             FAKESIG,
             EncodeData_FakeSigLen_);
}

static void
EncodeData_PrependDataTypeLength(TlvEncoder* en)
{
  struct rte_mbuf* m = TlvEncoder_AsMbuf(en);
  PrependVarNum(en, m->pkt_len);
  PrependVarNum(en, TtData);
}

void
EncodeData_(struct rte_mbuf* m,
            uint16_t namePrefixL,
            const uint8_t* namePrefixV,
            uint16_t nameSuffixL,
            const uint8_t* nameSuffixV,
            uint32_t freshnessPeriod,
            uint16_t contentL,
            const uint8_t* contentV)
{
  assert(rte_pktmbuf_headroom(m) >= EncodeData_GetHeadroom());
  assert(rte_pktmbuf_tailroom(m) >=
         EncodeData_GetTailroom(namePrefixL + nameSuffixL, contentL));

  TlvEncoder* en = MakeTlvEncoder(m);
  EncodeData_AppendNameNoSuffix(en, namePrefixL, namePrefixV, nameSuffixL);
  if (likely(nameSuffixL > 0)) {
    rte_memcpy(rte_pktmbuf_append(m, nameSuffixL), nameSuffixV, nameSuffixL);
  }
  EncodeData_AppendFreshnessContentSignature(
    en, freshnessPeriod, contentL, contentV);
  EncodeData_PrependDataTypeLength(en);
}

DataGen*
MakeDataGen_(struct rte_mbuf* m,
             uint16_t nameSuffixL,
             const uint8_t* nameSuffixV,
             uint32_t freshnessPeriod,
             uint16_t contentL,
             const uint8_t* contentV)
{
  assert(rte_pktmbuf_tailroom(m) >=
         DataGen_GetTailroom1(nameSuffixL, contentL));

  TlvEncoder* en = MakeTlvEncoder(m);
  if (nameSuffixL > 0) {
    rte_memcpy(rte_pktmbuf_append(m, nameSuffixL), nameSuffixV, nameSuffixL);
  }
  EncodeData_AppendFreshnessContentSignature(
    en, freshnessPeriod, contentL, contentV);

  m->vlan_tci = nameSuffixL;
  return (DataGen*)m;
}

void
DataGen_Close(DataGen* gen)
{
  rte_pktmbuf_free((struct rte_mbuf*)gen);
}

void
DataGen_Encode_(DataGen* gen,
                struct rte_mbuf* seg0,
                struct rte_mbuf* seg1,
                uint16_t namePrefixL,
                const uint8_t* namePrefixV)
{
  assert(rte_pktmbuf_tailroom(seg0) >= DataGen_GetTailroom0(namePrefixL));

  struct rte_mbuf* tailTpl = (struct rte_mbuf*)gen;
  uint16_t nameSuffixL = tailTpl->vlan_tci;
  rte_pktmbuf_attach(seg1, tailTpl);

  TlvEncoder* en = MakeTlvEncoder(seg0);
  EncodeData_AppendNameNoSuffix(en, namePrefixL, namePrefixV, nameSuffixL);
  rte_pktmbuf_chain(seg0, seg1);
  EncodeData_PrependDataTypeLength(en);
}
