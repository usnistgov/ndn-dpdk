# ndn-dpdk/container/strategycode

This package manages the BPF programs of NDN-DPDK's forwarding strategies.
Every loaded strategy is identified by a numeric ID and has a (non-unique) short name for presentation purposes.

## Strategy Loader

This package loads eBPF programs from an ELF object using two libraries: the DPDK BPF library and the [uBPF](https://github.com/iovisor/ubpf) library.
DPDK contains a BPF validator that disallows loops, which is necessary for a forwarding strategy.
uBPF has a limited ELF parser that is unable to handle ELF objects compiled by clang-4.0 and later.

To make these two libraries work together, `load.c` monkey-patches the `rte_bpf_load` function.
When loading a strategy ELF object:

1. The `Load` function writes the ELF object to a temporary file.
2. DPDK's `rte_bpf_elf_load` reads the file and processes the relocations.
3. The `rte_bpf_load` monkey patch receives eBPF instructions and passes them to uBPF.
4. A `struct ubpf_vm*` pointer is stored into the `bpf->prm.xsym` variable.
