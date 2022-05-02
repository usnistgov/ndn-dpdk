#include "cryptodev.h"

struct rte_cryptodev_sym_session*
CryptoDev_NewSha256DigestSession(struct rte_mempool* mp, uint8_t dev)
{
  struct rte_cryptodev_sym_session* sess = rte_cryptodev_sym_session_create(mp);
  if (unlikely(sess == NULL)) {
    return NULL;
  }

  struct rte_crypto_sym_xform xform = (struct rte_crypto_sym_xform){
    .type = RTE_CRYPTO_SYM_XFORM_AUTH,
    .auth.op = RTE_CRYPTO_AUTH_OP_GENERATE,
    .auth.algo = RTE_CRYPTO_AUTH_SHA256,
    .auth.digest_length = 32,
  };
  int res = rte_cryptodev_sym_session_init(dev, sess, &xform, mp);
  if (unlikely(res != 0)) {
    rte_errno = -res;
    rte_cryptodev_sym_session_free(sess);
  }
  return sess;
}
