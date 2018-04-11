# ndn-dpdk/strategy

This directory contains the [forwarder](../app/fwdp/)'s strategy API.
Code in this directory is compiled to BPF target using LLVM, except that `*_verify.c` is compiled with gcc to verify strategy API structs are declared same as forwarder's structs.

To implement a strategy named `foo`:

1.  Add `foo.c`, and include `api.h`.
2.  Implement the `SgMain` function as declared in `api.h`.
3.  All other functions must be `inline`.
4.  If necessary, spread other functions to `foo-*.h`.
