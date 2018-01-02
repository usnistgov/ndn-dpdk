#include "eth-tx.h"
#include "eth-face.h"

#include "../../core/logger.h"

#define LOG_PREFIX "(%" PRIu16 ",%" PRIu16 ") "
#define LOG_PARAM face->port, tx->queue

// max L2 burst size
static const int MAX_FRAMES = 64;

// max fragments per network layer packet
static const int MAX_FRAGMENTS = 16;

// minimum payload size per fragment
static const int MIN_PAYLOAD_SIZE_PER_FRAGMENT = 512;

// callback after NIC transmits packets
static uint16_t
EthTx_TxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                 uint16_t nPkts, void* face0)
{
  EthFace* face = (EthFace*)(face0);
  assert(queue == 0);
  EthTx* tx = &face->tx;

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    ++tx->nPkts[Packet_GetNdnPktType(pkt)];
    tx->nOctets += pkt->pkt_len;
  }

  return nPkts;
}

int
EthTx_Init(EthFace* face, EthTx* tx)
{
  assert(tx->indirectMp != NULL);
  assert(tx->headerMp != NULL);
  assert(rte_pktmbuf_data_room_size(tx->headerMp) >=
         EthTx_GetHeaderMempoolDataRoom());

  uint16_t mtu;
  int res = rte_eth_dev_get_mtu(face->port, &mtu);
  if (res != 0) {
    return -res;
  }
  const int maxLpHeaderSize =
    EncodeLpHeaders_GetHeadroom() + EncodeLpHeaders_GetTailroom();
  int fragmentPayloadSize = (int)mtu - maxLpHeaderSize;
  if (fragmentPayloadSize < MIN_PAYLOAD_SIZE_PER_FRAGMENT) {
    return false;
  }
  tx->fragmentPayloadSize = (uint16_t)fragmentPayloadSize;

  rte_eth_macaddr_get(face->port, &tx->ethhdr.s_addr);
  const uint8_t dstAddr[] = { NDN_ETHER_MCAST };
  rte_memcpy(&tx->ethhdr.d_addr, dstAddr, sizeof(tx->ethhdr.d_addr));
  tx->ethhdr.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  tx->__txCallback =
    rte_eth_add_tx_callback(face->port, tx->queue, &EthTx_TxCallback, face);
  if (tx->__txCallback == NULL) {
    return rte_errno;
  }

  return 0;
}

void
EthTx_Close(EthFace* face, EthTx* tx)
{
  rte_eth_remove_tx_callback(face->port, tx->queue, tx->__txCallback);
  tx->__txCallback = NULL;
}

enum EthTx_FragmentErr
{
  // Fragmentation failed but burst processing should continue
  EthTx_FragmentErr_Continue = -1,
  // Fragmentation failed but burst processing should stop
  EthTx_FragmentErr_Stop = -2,
};

// Fragment L3 packet into NDNLP packets filled in fragments[0..retval-1].
// fragments[i] has NDNLP header chained with payload, but not Ethernet header.
// Returns number of fragments created, or EthTx_FragmentErr on failure.
static inline int
EthTx_Fragment(EthFace* face, EthTx* tx, struct rte_mbuf* pkt,
               struct rte_mbuf* fragments[MAX_FRAGMENTS])
{
  assert(pkt->pkt_len > 0);
  int nFragments = pkt->pkt_len / tx->fragmentPayloadSize +
                   (int)(pkt->pkt_len % tx->fragmentPayloadSize > 0);
  if (unlikely(nFragments > MAX_FRAGMENTS)) {
    ++tx->nL3OverLength;
    return EthTx_FragmentErr_Continue;
  }

  int res = rte_pktmbuf_alloc_bulk(tx->headerMp, fragments, nFragments);
  if (unlikely(res != 0)) {
    ++tx->nAllocFails;
    return 0;
  }

  MbufLoc pos;
  MbufLoc_Init(&pos, pkt);

  LpPkt lpp;
  if (Packet_GetL2PktType(pkt) == L2PktType_NdnlpV2) {
    rte_memcpy(&lpp, Packet_GetLpHdr(pkt), sizeof(lpp));
  } else {
    memset(&lpp, 0, sizeof(lpp));
  }

  for (int i = 0; i < nFragments; ++i) {
    uint32_t fragSize = tx->fragmentPayloadSize;
    if (fragSize > pos.rem) {
      fragSize = pos.rem;
    }
    struct rte_mbuf* payload =
      MbufLoc_MakeIndirect(&pos, fragSize, tx->indirectMp);
    if (unlikely(payload == NULL)) {
      assert(rte_errno == ENOENT);
      ++tx->nAllocFails;
      FreeMbufs(fragments, nFragments);
      return EthTx_FragmentErr_Stop;
    }
    MbufLoc_Init(&lpp.payload, payload);

    lpp.seqNo = ++tx->lastSeqNo;
    lpp.fragIndex = (uint16_t)i;
    lpp.fragCount = (uint16_t)nFragments;

    fragments[i]->data_off =
      sizeof(struct ether_hdr) + EncodeLpHeaders_GetHeadroom();
    EncodeLpHeaders(fragments[i], &lpp);
    res = rte_pktmbuf_chain(fragments[i], payload);
    if (unlikely(res != 0)) {
      ++tx->nL3OverLength;
      FreeMbufs(fragments, nFragments);
      return EthTx_FragmentErr_Continue;
    }
  }

  return nFragments;
}

static inline void
EthTx_SendFrames(EthFace* face, EthTx* tx, struct rte_mbuf** frames,
                 uint16_t nFrames)
{
  ++tx->nL2Bursts;

  uint16_t nSent = rte_eth_tx_burst(face->port, tx->queue, frames, nFrames);
  if (nSent == nFrames) {
    return;
  }

  ++tx->nL2Incomplete;
  FreeMbufs(frames + nSent, nFrames - nSent);
}

void
EthTx_TxBurst(EthFace* face, EthTx* tx, struct rte_mbuf** pkts, uint16_t nPkts)
{
  ++tx->nL3Bursts;
  struct rte_mbuf* frames[MAX_FRAMES + MAX_FRAGMENTS];
  int nFrames = 0;

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    int nFragments = EthTx_Fragment(face, tx, pkt, frames + nFrames);
    if (unlikely(nFragments <= 0)) {
      if (nFragments == EthTx_FragmentErr_Continue) {
        continue;
      } else if (nFragments == EthTx_FragmentErr_Stop) {
        break;
      } else {
        assert(false);
      }
    }

    NdnPktType l3type =
      Packet_GetNdnPktType(pkt); // first fragment has real L3 type
    for (int last = nFrames + nFragments; nFrames < last; ++nFrames) {
      struct rte_mbuf* frame = frames[nFrames];
      Packet_SetNdnPktType(frame, l3type);
      l3type =
        NdnPktType_None; // subsequent fragment has None to count as L2 only

      struct ether_hdr* eth =
        (struct ether_hdr*)rte_pktmbuf_prepend(frame, sizeof(struct ether_hdr));
      assert(eth != NULL);
      rte_memcpy(eth, &tx->ethhdr, sizeof(*eth));
    }

    while (unlikely(nFrames >= MAX_FRAMES)) {
      EthTx_SendFrames(face, tx, frames, MAX_FRAMES);
#if MAX_FRAGMENTS > MAX_FRAMES
#define MoveUpFragments memmove
#else // nFragments is no more than MAX_FRAME so no overlapping
#define MoveUpFragments rte_memcpy
#endif
      MoveUpFragments(frames, frames + MAX_FRAMES,
                      sizeof(frames[0]) * nFragments);
#undef MoveUpFragments
      nFrames -= MAX_FRAMES;
    }
  }

  if (likely(nFrames > 0)) {
    EthTx_SendFrames(face, tx, frames, nFrames);
  }
}

void
EthTx_ReadCounters(EthFace* face, EthTx* tx, FaceCounters* cnt)
{
  cnt->txl3.nInterests = tx->nPkts[NdnPktType_Interest];
  cnt->txl3.nData = tx->nPkts[NdnPktType_Data];
  cnt->txl3.nNacks = tx->nPkts[NdnPktType_Nack];

  cnt->txl2.nFrames = tx->nPkts[NdnPktType_None] + cnt->txl3.nInterests +
                      cnt->txl3.nData + cnt->txl3.nNacks;
  cnt->txl2.nOctets = tx->nOctets;
}
