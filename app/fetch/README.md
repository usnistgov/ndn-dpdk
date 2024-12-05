# ndn-dpdk/app/fetch

This package is the congestion aware fetcher, used in the [traffic generator](../tg).
It implements a consumer that follows the TCP CUBIC congestion control algorithm, simulating traffic patterns similar to bulk file transfer.
It requires at least one thread, running the `FetchThread_Run` function.

## Fetch Task Definition

**TaskDef** defines a fetch task that retrieves one segmented object.
A *segmented object* is a group of NDN packets, which have a common name prefix and have SegmentNameComponent as the last component.
The TaskDef contains these fields:

* Prefix: a name prefix except the last SegmentNameComponent.
  * Importantly, if you are retrieving from the [file server](../fileserver), this field must end with the VersionNameComponent.
* InterestLifetime
* HopLimit
* SegmentRange: retrieve a consecutive subset of the available segments.
  * If the fetcher encounters a Data packet whose FinalBlockId equals its last name component, the fetching will terminate at this segment, even if the upper bound of SegmentRange has not been reached.

Normally, a fetch task generates traffic similar to bulk file transfer, in which contents of the received packets are discarded.
It is however possible to write the received payload into a file.
In this case, the TaskDef additionally contains these fields:

* Filename: output file name.
* FileSize: total file size.
* SegmentLen: the payload length in every segment; the last segment may be shorter.

## Fetcher and its Workers

A **worker** is a thread running the `FetchThread_Run` function.
It can simultaneously process zero or more fetch tasks, which are arranged in an RCU-protected linked list.
It has an io\_uring handle in order to write payload to files when requested.

A **TaskContext** stores information of an ongoing fetch task, which can be initialized from a TaskDef.
It includes a **taskSlot** (aka **C.FetchTask**) used by C code, along with several Go objects.
It is responsible for opening and closing the file, if the TaskDef requests to write payload to a file.
Each taskSlot has an index number that used as the PIT token for its Interests, which allows the reply Data packets to come back to the same taskSlot.

**FetchLogic** contained with the taskSlot implements the algorithmic part of the fetch procedure.
It includes an RTT estimator, a CUBIC-like congestion control implementation, and a retransmission queue.
It makes decisions on when to transmit an Interest for a certain segment number, and gets notified about when the Data arrives with or without a congestion mark.
Nack packets are not considered in the congestion aware fetcher.

**Fetcher** is the top level.
It controls one or more workers and owns one or more task slots.
A incoming TaskDef is placed into an unused task slot, and then added to the worker with the least number of ongoing fetch tasks.
