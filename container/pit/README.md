# ndn-dpdk/container/pit

This package implements the **Pending Interest Table (PIT)**.

## Structure

PIT is part of the [PIT-CS Composite Table (PCCT)](../pcct/).
PCCT provides storage and lookup functions for PIT.

The **PIT token** is actually the 48-bit token identifying a `PccEntry`.
PIT automatically adds and removes this token upon inserting and deleting PIT entries on a `PccEntry`.
Since each `PccEntry` can contain up to two PIT entries (one for MustBeFresh=0 and one for MustBeFresh=1), the same token identifies both PIT entries.

## PIT Entry

Each **PIT entry** contains some **PIT downstream records** (`PitDn` type) and **PIT upstream records** (`PitUp`).
It also a representative Interest, a [timer](../mintmr/), and several other fields that are aggregated from DN and UP records.

The `PitEntry` type directly stores a small number of `PitDn` and `PitUp`.
If a PIT entry needs more DN or UP records, the PIT extends the `PitEntry` with more DN and UP slots with a `PitEntryExt`, allocated from PCCT's mempool.
