# ndn-dpdk/container/pit

This package implements the **Pending Interest Table (PIT)**.

## Structure

PIT is part of the [PIT-CS Composite Table (PCCT)](../pcct/).
PCCT provides storage and lookup functions for PIT.

The **PIT token** is actually the 48-bit token identifying a `PccEntry`.
PIT automatically adds and removes this token upon inserting and deleting PIT entries on a `PccEntry`.
Since each `PccEntry` can contain up to two PIT entries (one for MustBeFresh=0 and one for MustBeFresh=1), the same token identifies both PIT entries.
