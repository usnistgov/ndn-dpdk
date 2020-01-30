# ndn-dpdk/app/fetch

This package is part of the [packet generator](../ping/).
It implements a consumer that follows TCP CUBIC congestion control algorithm, simulating traffic patterns similar to bulk file transfer.
It runs `Fetcher_Run` function in a *fetcher thread* ("CLIR" role) of the traffic generator.
