# ndn-dpdk/container/ndt

This package implements the **Name Dispatch Table (NDT)**.

In a forwarder with a partitioned PIT, the NDT decides which PIT partition an Interest belongs to.
One or more dispatcher threads query the NDT using the names of the incoming Interests, and then dispatch each Interest to the forwarding thread that handles the chosen PIT partition.

This implementation chooses the correct PIT partition with the following algorithm:

1. Compute the hash of the name's first *prefixLen* components. If the name has fewer than *prefixLen* components, use all components.
2. Truncate the hash to the last *indexBits* bits.
3. Lookup the table using the truncated hash. The table entry indicates the chosen PIT partition.

The NDT maintains counters of how many times each table entry has been selected.
With these counters, a maintenance thread can periodically reconfigure the NDT to balance the load among the available forwarding threads.
