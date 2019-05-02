# ndn-dpdk/core/running_stat

This package implements [Knuth and Welford method for computing standard deviation](https://www.johndcook.com/blog/skewness_kurtosis/).
Compared to the original `RunningStats` C++ class, this implementation removes skewness and kurtosis support, tracks minimum and maximum, and can be configured to periodically sample the input instead of sampling every input.
To add an input, use C function `RunningStat_Push`, or `RunningStat_Push1` when minimum and maximum are unnecessary.
To compute average and standard deviation, use Go type `RunningStat`.
