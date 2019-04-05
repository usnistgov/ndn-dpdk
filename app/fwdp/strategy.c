#include "strategy.h"

const struct rte_bpf_xsym*
SgGetXsyms(int* nXsyms)
{
  static const struct rte_bpf_xsym xsyms[] = {
    {
      0, .name = "SgForwardInterest", .type = RTE_BPF_XTYPE_FUNC,
      .func =
        {
          .val = (void*)SgForwardInterest,
          .nb_args = 2,
          .args =
            {
                [0] =
                  {
                    .type = RTE_BPF_ARG_PTR, .size = sizeof(SgCtx),
                  },
                [1] =
                  {
                    .type = RTE_BPF_ARG_RAW,
                  },
            },
        },
    },
  };
  *nXsyms = RTE_DIM(xsyms);
  return xsyms;
}
