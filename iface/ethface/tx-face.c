#include "tx-face.h"
#include "../../core/logger.h"

// max L2 burst size
static const int MAX_FRAMES = 64;

// max fragments per network layer packet
static const int MAX_FRAGMENTS = 16;

// minimum payload size per fragment
static const int MIN_PAYLOAD_SIZE_PER_FRAGMENT = 512;

// callback after NIC transmits packets
static uint16_t
EthTxFace_TxCallback(uint16_t port, uint16_t queue, struct rte_mbuf** pkts,
                     uint16_t nPkts, void* face0)
{
  EthTxFace* face = (EthTxFace*)(face0);

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    ++face->nPkts[Packet_GetNdnPktType(pkt)];
    face->nOctets += pkt->pkt_len;
  }

  return nPkts;
}

bool
EthTxFace_Init(EthTxFace* face)
{
  assert(face->indirectMp != NULL);
  assert(face->headerMp != NULL);
  assert(rte_pktmbuf_data_room_size(face->headerMp) >=
         EthTxFace_GetHeaderMempoolDataRoom());

  uint16_t mtu;
  int res = rte_eth_dev_get_mtu(face->port, &mtu);
  if (res != 0) {
    return false;
  }
  const int maxLpHeaderSize =
    EncodeLpHeaders_GetHeadroom() + EncodeLpHeaders_GetTailroom();
  int fragmentPayloadSize = (int)mtu - maxLpHeaderSize;
  if (fragmentPayloadSize < MIN_PAYLOAD_SIZE_PER_FRAGMENT) {
    return false;
  }
  face->fragmentPayloadSize = (uint16_t)fragmentPayloadSize;

  rte_eth_macaddr_get(face->port, &face->ethhdr.s_addr);
  const uint8_t dstAddr[] = { NDN_ETHER_MCAST };
  rte_memcpy(&face->ethhdr.d_addr, dstAddr, sizeof(face->ethhdr.d_addr));
  face->ethhdr.ether_type = rte_cpu_to_be_16(NDN_ETHERTYPE);

  face->__txCallback = rte_eth_add_tx_callback(face->port, face->queue,
                                               &EthTxFace_TxCallback, face);
  if (face->__txCallback == NULL) {
    return false;
  }

  return true;
}

void
EthTxFace_Close(EthTxFace* face)
{
  rte_eth_remove_tx_callback(face->port, face->queue, face->__txCallback);
  face->__txCallback = NULL;
}

enum EthTxFace_FragmentErr
{
  // Fragmentation failed but burst processing should continue
  EthTxFace_FragmentErr_Continue = -1,
  // Fragmentation failed but burst processing should stop
  EthTxFace_FragmentErr_Stop = -2,
};

// Fragment L3 packet into NDNLP packets filled in fragments[0..retval-1].
// fragments[i] has NDNLP header chained with payload, but not Ethernet header.
// Returns number of fragments created, or EthTxFace_FragmentErr on failure.
static inline int
EthTxFace_Fragment(EthTxFace* face, struct rte_mbuf* pkt,
                   struct rte_mbuf* fragments[MAX_FRAGMENTS])
{
  assert(pkt->pkt_len > 0);
  int nFragments = pkt->pkt_len / face->fragmentPayloadSize +
                   (int)(pkt->pkt_len % face->fragmentPayloadSize > 0);
  if (unlikely(nFragments > MAX_FRAGMENTS)) {
    ++face->nL3OverLength;
    return EthTxFace_FragmentErr_Continue;
  }

  int res = rte_pktmbuf_alloc_bulk(face->headerMp, fragments, nFragments);
  if (unlikely(res != 0)) {
    ++face->nAllocFails;
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
    uint32_t fragSize = face->fragmentPayloadSize;
    if (fragSize > pos.rem) {
      fragSize = pos.rem;
    }
    struct rte_mbuf* payload =
      MbufLoc_MakeIndirect(&pos, fragSize, face->indirectMp);
    if (unlikely(payload == NULL)) {
      assert(rte_errno == ENOENT);
      ++face->nAllocFails;
      FreeMbufs(fragments, nFragments);
      return EthTxFace_FragmentErr_Stop;
    }
    MbufLoc_Init(&lpp.payload, payload);

    lpp.seqNo = ++face->lastSeqNo;
    lpp.fragIndex = (uint16_t)i;
    lpp.fragCount = (uint16_t)nFragments;

    fragments[i]->data_off =
      sizeof(struct ether_hdr) + EncodeLpHeaders_GetHeadroom();
    EncodeLpHeaders(fragments[i], &lpp);
    res = rte_pktmbuf_chain(fragments[i], payload);
    if (unlikely(res != 0)) {
      ++face->nL3OverLength;
      FreeMbufs(fragments, nFragments);
      return EthTxFace_FragmentErr_Continue;
    }
  }

  return nFragments;
}

static inline void
EthTxFace_SendFrames(EthTxFace* face, struct rte_mbuf** frames,
                     uint16_t nFrames)
{
  ++face->nL2Bursts;

  uint16_t nSent = rte_eth_tx_burst(face->port, face->queue, frames, nFrames);
  if (nSent == nFrames) {
    return;
  }

  ++face->nL2Incomplete;
  FreeMbufs(frames + nSent, nFrames - nSent);
}

void
EthTxFace_TxBurst(EthTxFace* face, struct rte_mbuf** pkts, uint16_t nPkts)
{
  ++face->nL3Bursts;
  struct rte_mbuf* frames[MAX_FRAMES + MAX_FRAGMENTS];
  int nFrames = 0;

  for (uint16_t i = 0; i < nPkts; ++i) {
    struct rte_mbuf* pkt = pkts[i];
    int nFragments = EthTxFace_Fragment(face, pkt, frames + nFrames);
    if (unlikely(nFragments <= 0)) {
      if (nFragments == EthTxFace_FragmentErr_Continue) {
        continue;
      } else if (nFragments == EthTxFace_FragmentErr_Stop) {
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
      rte_memcpy(eth, &face->ethhdr, sizeof(*eth));
    }

    while (unlikely(nFrames >= MAX_FRAMES)) {
      EthTxFace_SendFrames(face, frames, MAX_FRAMES);
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
    EthTxFace_SendFrames(face, frames, nFrames);
  }
}