#ifndef NDNDPDK_PDUMP_FORMAT_H
#define NDNDPDK_PDUMP_FORMAT_H

/** @file */

#include "../core/common.h"
#include "enum.h"
#include <pcap/pcap.h>
#include <pcap/sll.h>

/**
 * @brief DLT_LINUX_SLL direction constants in network byte order.
 *
 * Each value is rte_be16_t type, which has same size as sll_pkttype.
 */
enum
{
  SLLIncoming = RTE_BE16(LINUX_SLL_HOST),
  SLLOutgoing = RTE_BE16(LINUX_SLL_OUTGOING),
};

/** @brief PCAPNG interface description block header. */
typedef struct PcapngIDB
{
  rte_le32_t blockType;
  rte_le32_t totalLength;
  rte_le16_t linkType;
  rte_le16_t reserved;
  rte_le32_t snaplen;
} __rte_packed PcapngIDB;

/** @brief PCAPNG enhanced packet block header. */
typedef struct PcapngEPB
{
  rte_le32_t blockType;
  rte_le32_t totalLength;
  rte_le32_t intf;
  rte_le32_t timeHi;
  rte_le32_t timeLo;
  rte_le32_t capLen;
  rte_le32_t origLen;
} __rte_packed PcapngEPB;

/** @brief PCAPNG enhanced packet block header and tcpdump DLT_LINUX_SLL header. */
typedef struct PcapngEPBSLL
{
  PcapngEPB epb;
  struct sll_header sll;
} __rte_packed PcapngEPBSLL;
static_assert(sizeof(PcapngEPBSLL) == sizeof(PcapngEPB) + SLL_HDR_LEN, "");

#endif // NDNDPDK_PDUMP_FORMAT_H
