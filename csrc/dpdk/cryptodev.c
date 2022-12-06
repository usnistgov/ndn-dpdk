#include "cryptodev.h"

struct rte_crypto_sym_xform CryptoDev_Sha256Xform = {
  .type = RTE_CRYPTO_SYM_XFORM_AUTH,
  .auth.op = RTE_CRYPTO_AUTH_OP_GENERATE,
  .auth.algo = RTE_CRYPTO_AUTH_SHA256,
  .auth.digest_length = 32,
};
