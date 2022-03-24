# ndn-dpdk/bpf/strategy

This directory contains forwarding strategies, implemented using the strategy API defined in [`api.h`](../../csrc/strategyapi/api.h).

To implement a strategy named `foo`:

1. Add `foo.c`, and include `api.h`.
2. Implement the `SgMain` function as declared in `api.h`.
3. If the strategy accepts JSON parameters, implement the `SgInit` function and provide a JSON schema via `SGJSON_SCHEMA` macro.
4. All other functions must be marked as `SUBROUTINE`.
