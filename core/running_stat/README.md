# ndn-dpdk/core/running_stat

This package implements [accurate running variance computation](https://www.johndcook.com/blog/standard_deviation/).
Compared to the original `RunningStat` C++ class, this implementation tracks minimum and maximum, and can be configured to periodically sample the input instead of sampling every input.
To add an input, use C function `RunningStat_Push`, or `RunningStat_Push1` when minimum and maximum are unnecessary.
To compute average and standard deviation, use Go type `RunningStat`.
