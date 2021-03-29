# ndn-dpdk/bpf/strategy

This directory contains forwarding strategies, implemented using the strategy API defined in [`api.h`](../../csrc/strategyapi/api.h).

To implement a strategy named `foo`:

1. Add `foo.c`, and include `api.h`.
2. Implement the `SgMain` function as declared in `api.h`.
3. All other functions must be `inline` (use `SUBROUTINE` macro).
4. If necessary, spread other functions to `foo-*.h`.
