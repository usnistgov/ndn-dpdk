# ndn-dpdk/core/runningstat

This package implements [Knuth and Welford's method for computing the standard deviation](https://www.johndcook.com/blog/skewness_kurtosis/).
Compared to the original `RunningStats` C++ class, this implementation does not support skewness and kurtosis, adds tracking of the minimum and maximum values, and can be configured to periodically sample the input instead of sampling every input.

To add an input, use the C function `RunningStat_Push`, or `RunningStat_Push1` when minimum and maximum are unnecessary.
To compute the average and standard deviation, use the Go type `RunningStat`.
