#include "source.h"
#include "parse.h"

void
PdumpSource_Process(PdumpSource* s, struct rte_mbuf** pkts, uint16_t count) {
  struct rte_mbuf* output[MaxBurstSize];
  uint16_t nOutput = 0;

  for (uint16_t i = 0; i < count; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    if (s->filter != NULL && !s->filter(s, pkt)) {
      continue;
    }

    if (s->mbufCopy) {
      pkt = rte_pktmbuf_copy(pkt, s->directMp, 0, UINT32_MAX);
      if (unlikely(pkt == NULL)) {
        break;
      }
    }

    pkt->port = s->mbufPort;
    pkt->packet_type = s->mbufType;
    output[nOutput++] = pkt;
  }

  Mbuf_EnqueueVector(output, nOutput, s->queue, true);
}

__attribute__((nonnull(1))) PdumpSource*
PdumpSourceRef_Set(PdumpSourceRef* ref, PdumpSource* s) {
  return rcu_xchg_pointer(&ref->s, s);
}

__attribute__((nonnull)) static __rte_always_inline uint32_t
PdumpFaceSource_NameProb(const PdumpFaceSource* source, struct rte_mbuf* pkt) {
  if (source->nameL[0] == 0) {
    return source->sample[0];
  }

  LName name = Pdump_ExtractName(pkt);
  if (unlikely(name.length == 0)) {
    return 0;
  }

  int index = LNamePrefixFilter_Find(name, PdumpMaxNames, source->nameL, source->nameV);
  return index >= 0 ? source->sample[index] : 0;
}

bool
PdumpFaceSource_Filter(PdumpSource* s0, struct rte_mbuf* pkt) {
  PdumpFaceSource* s = container_of(s0, PdumpFaceSource, base);
  uint32_t prob = PdumpFaceSource_NameProb(s, pkt);
  return prob > 0 &&                      // skip pcg32 computation when there's no name match
         prob >= pcg32_random_r(&s->rng); // '>=' because UINT32_MAX means always
}
