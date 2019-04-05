# ndn-dpdk/container/strategycode

This package organizes BPF programs of forwarding strategies.
Every loaded strategy is identified with a numeric ID, and has a (non-unique) short name for description.

## Loader

This package loads an eBPF program from an ELF object using two libraries: DPDK BPF library and uBPF library.
DPDK contains a BPF validator that disallows loops, which is necessary for a forwarding strategy.
uBPF has a limited ELF parser that is unable to handle ELF objects compiled by clang-4.0 and later.

To make these two libraries work together, `load.c` monkey-patches `rte_bpf_load` function.
When loading a strategy ELF object:

1. Go `Load` function writes the ELF object to a temporary file.
2. DPDK `rte_bpf_elf_load` reads the file and processes relocations.
3. The `rte_bpf_load` monkey patch receives eBPF instructions and passes them to uBPF.
4. `struct ubpf_vm*` pointer is stored at `bpf->prm.xsym` variable.
