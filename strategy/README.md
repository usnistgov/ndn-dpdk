# ndn-dpdk/strategy

This directory contains the [forwarder](../app/fwdp/)'s strategy API.
Code in this directory is compiled to BPF target using LLVM.

To implement a strategy named `foo`:

1.  Add `foo.c`, and include `api.h`.
2.  Implement the `Program` function, as declared in `api.h`.
3.  All other functions must be `inline`.
4.  If necessary, spread other functions to `foo-*.h`.
