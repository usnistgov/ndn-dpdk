#include "cryptodev.h"

struct rte_crypto_sym_xform theSha256DigestXform;

RTE_INIT(InitSha256DigestXform)
{
  theSha256DigestXform = (const struct rte_crypto_sym_xform){ 0 };
  theSha256DigestXform.type = RTE_CRYPTO_SYM_XFORM_AUTH;
  theSha256DigestXform.auth.op = RTE_CRYPTO_AUTH_OP_GENERATE;
  theSha256DigestXform.auth.algo = RTE_CRYPTO_AUTH_SHA256;
  theSha256DigestXform.auth.digest_length = 32;
}
