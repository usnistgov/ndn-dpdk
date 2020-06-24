# ndn-dpdk/app/fetch

This package is part of the [packet generator](../ping).
It implements a consumer that follows the TCP CUBIC congestion control algorithm, simulating traffic patterns similar to bulk file transfer.
It runs the `FetchThread_Run` function in a *fetcher thread* ("CLIR" role) of the traffic generator.
