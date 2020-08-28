# ndn-dpdk/container/ndt

This package implements the **Name Dispatch Table (NDT)**.

The NDN-DPDK forwarder uses a sharded PIT structure, where each forwarding thread owns a shard of the PIT.
The NDT decides which PIT shard an Interest belongs to.
One or more input threads query the NDT using the names of the incoming Interests, and then dispatch each Interest to the forwarding thread that handles the chosen PIT shard.

In this implementation, the NDT is a linear array of *2^B* entries, where each entry indicates a PIT shard number.
Each NUMA socket has a replica of this array, so that it can be accessed efficiently.

This implementation chooses the correct PIT shard with the following algorithm:

1. Compute the hash of the name's first *prefixLen* components. If the name has fewer than *prefixLen* components, use all components.
2. Truncate the hash to the last *B* bits.
3. Lookup the table using the truncated hash. The table entry indicates the chosen PIT shard.

The NDT maintains counters of how many times each table entry has been selected.
With these counters, a maintenance thread (not yet implemented in this codebase) can periodically reconfigure the NDT to balance the load among the available forwarding threads.
