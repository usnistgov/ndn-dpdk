# ndn-dpdk/container/pit

This package implements the **Pending Interest Table (PIT)**.

## Structure

The PIT is part of the [PIT-CS Composite Table (PCCT)](../pcct).
The PCCT provides the underlying storage and lookup functions for the PIT.

The **PIT token** is actually the 48-bit token identifying a PCC entry.
The PIT automatically adds and removes this token upon inserting and deleting PIT entries on a PCC entry.
Since each PCC entry can contain up to two PIT entries (one for *MustBeFresh=0* and one for *MustBeFresh=1*), the same token identifies both PIT entries.

## PIT Entry

Each **PIT entry** can contain many *PIT downstream records* (`PitDn` type) and *PIT upstream records* (`PitUp` type).
The `PitEntry` type directly stores a small number of `PitDn` and `PitUp`.
If more downstream/upstream records are required, the PIT extends the `PitEntry` with additional DN and UP slots using a `PitEntryExt` allocated from the PCCT's mempool.

A PIT entry also contains:

* a representative Interest
* a [timer](../mintmr)
* several other fields aggregated from downstream and upstream records
* a "FIB reference" that allows efficient access to the associated FIB entry (`PitEntry_FindFibEntry`)
