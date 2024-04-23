# ndn-dpdk/app/fetch

This package is the congestion aware fetcher, used in the [traffic generator](../tg).
It implements a consumer that follows the TCP CUBIC congestion control algorithm, simulating traffic patterns similar to bulk file transfer.
It requires at least one thread, running the `FetchThread_Run` function.
