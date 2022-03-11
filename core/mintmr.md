# ndn-dpdk/core MinTmr

`mintmr.h` implements a **minute scheduler (MinTmr)**.
It allows scheduling events within the next few minutes, and triggering them at an accuracy on the order of milliseconds.

## Usage

A typical use case of the minute scheduler is to cleanup inactive table entries.
This section describes how to use the minute scheduler in the context of this use case.

1. Embed a `MinTmr` struct in the table entry.
2. Create a `MinSched` instance using the `MinSched_New` constructor.
3. Arm a timer with the `MinTmr_After` function.
4. Invoke `MinSched_Trigger` periodically from the main loop.
   The callback function specified in the constructor will be invoked for each timer that has expired.
   The callback function can recover the table entry from the `MinTmr*` via the `container_of` macro and then deallocate the entry.

## Design

Most generic timer libraries, including DPDK's `rte_timer.h`:

* can schedule events any time into the future;
* maintain an ordered list of scheduled events;
* record the callback function along with each scheduled event.

The minute scheduler takes a different approach.

1. The `MinSched` instance has a number of slots.
   Each slot contains timers that should expire at the same time, organized in a doubly linked list.
2. The `MinTmr_After` function selects a slot that a timer belongs to, and inserts the timer into that slot.
3. The `MinSched_Trigger` function checks whether timers in the next slot are expiring.
   If so, it invokes the callback function on each timer.

The minute scheduler is faster than generic timer libraries because it does not maintain an ordered list of scheduled events, but simply puts them into a slot within an array.
It also consumes less memory because it records only one callback function for all events instead of one for each event.
It has the limitation of not being able to schedule an event at an arbitrary time in the future because there is a fixed number of slots in the array.

The number of slots and the interval of each slot are specified in the `MinSched_New` constructor.
They affect how far in the future events can be scheduled.
For example, setting 32 slots and a 100 ms interval allows scheduling at most 3100 ms into the future.
