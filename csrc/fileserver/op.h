#ifndef NDNDPDK_FILESERVER_OP_H
#define NDNDPDK_FILESERVER_OP_H

/** @file */

#include "../ndni/lp.h"
#include "../ndni/name.h"
#include "enum.h"

typedef struct FileServerFd FileServerFd;

/** @brief File server readv operation. */
typedef struct FileServerOp
{
  FileServerFd* fd;
  uint64_t segment;
  struct rte_mbuf* data;
  int iovcnt;
  uint16_t contentLen;
  LpL3 interestL3;
  struct iovec iov[LpMaxFragments];
} FileServerOp;

/**
 * @brief Sign and transmit a Data packet.
 * @param p @c FileServer* .
 * @param ctx @c RxBurstCtx* or @c TxBurstCtx* .
 * @param fd @c FileServerFd* .
 * @param func function name string.
 * @param dataPkt Data mbuf.
 * @param interestL3 Interest @c LpL3 value.
 * @return Data @c Packet* .
 */
#define FileServer_SignAndSend(p, ctx, fd, func, dataPkt, interestL3)                              \
  __extension__({                                                                                  \
    Packet* dataNpkt = DataEnc_Sign((dataPkt), &(p)->mp, Face_PacketTxAlign((p)->face));           \
    if (unlikely(dataNpkt == NULL)) {                                                              \
      N_LOGW(func " fd=%d drop=data-sign-err", (fd)->fd);                                          \
    } else {                                                                                       \
      Mbuf_SetTimestamp((dataPkt), (ctx)->now);                                                    \
      *Packet_GetLpL3Hdr(dataNpkt) = (interestL3);                                                 \
      (ctx)->data[(ctx)->nData++] = dataNpkt;                                                      \
    }                                                                                              \
    dataNpkt;                                                                                      \
  })

#endif // NDNDPDK_FILESERVER_OP_H
