# ndn-dpdk/container/ndt

This package implements the **Name Dispatch Table (NDT)**.

In a forwarder with partitioned PIT, the NDT decides which PIT partition an Interest belongs to.
One or more dispatcher threads queries the NDT using incoming Interest names, and then dispatches the Interest to the correct forwarding thread that handles the chosen PIT partition.

This implementation chooses PIT partition with the following algorithm:

1. Compute hash of the name's first *prefixLen* components. If the name has fewer than *prefixLen* components, use all components.
2. Truncate the hash to the last *indexBits* bits.
3. Lookup the table using the truncated hash. The table entry indicates the chosen PIT partition.

The NDT maintains counters on how many times each table entry is selected.
With these counters, a maintainance thread can adjust the NDT to balance the load of forwarding threads.
