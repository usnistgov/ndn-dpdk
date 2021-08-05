#include "face.h"
#include "parse.h"

__attribute__((nonnull)) static inline uint32_t
NameFilter(PdumpFace* pd, struct rte_mbuf* pkt)
{
  if (pd->nameL[0] == 0) {
    return pd->sample[0];
  }

  LName name = Pdump_ExtractName(pkt);
  if (name.length == 0) {
    return 0;
  }

  static_assert(sizeof(pd->nameV) <= UINT16_MAX, "");
  uint16_t offset = 0;
  for (int i = 0; i < PdumpMaxNames; ++i) {
    if (pd->nameL[i] == 0) {
      break;
    }

    LName prefix = {
      .value = RTE_PTR_ADD(pd->nameV, offset),
      .length = pd->nameL[i],
    };
    if (LName_IsPrefix(prefix, name) >= 0) {
      return pd->sample[i];
    }
    offset += prefix.length;
  }
  return 0;
}

void
PdumpFace_Process(PdumpFace* pd, FaceID id, struct rte_mbuf** pkts, uint16_t count)
{
  struct rte_mbuf* dumped[MaxBurstSize];
  uint16_t nDumped = 0;

  for (uint16_t i = 0; i < count; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    uint32_t sample = NameFilter(pd, pkt);
    fflush(stdout);
    if (sample == 0 || sample < pcg32_random_r(&pd->rng)) {
      continue;
    }

    struct rte_mbuf* copy = rte_pktmbuf_copy(pkt, pd->directMp, 0, UINT32_MAX);
    if (unlikely(copy == NULL)) {
      break;
    }
    copy->port = id;
    copy->packet_type = pd->sllType;
    dumped[nDumped++] = copy;
  }

  Mbuf_EnqueueVector(dumped, nDumped, pd->queue);
}
