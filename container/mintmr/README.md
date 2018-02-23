# ndn-dpdk/container/mintmr

This directory implements a **minute scheduler**.
It allows scheduling events within the next few minutes, and triggering them at 100ms accuracy.

## Usage

A typical use case of minute scheduler is to clean out inactive table entries.
This section describes how to use the minute scheduler in the context of this use case.

1. Embed a `MinTmr` struct in the table entry.
2. Create a `MinSched` instance with `MinSched_New` constructor.
3. Set the timer with `MinTmr_After` function.
4. Invoke `MinSched_Trigger` periodically from the main loop. The callback function specified in the constructor will be invoked for each timer that has expired. The callback function can recover the table entry from the `MinTmr*` via `container_of` macro, and then deallocate the table entry.

## Design

Most generic timer libraries, including DPDK `rte_timer.h`:

* can schedule events any time into the future.
* maintain an ordered list of scheduled events.
* record the callback function along with each scheduled events.

The minute scheduler takes a different approach.

1. The `MinSched` instance has a number of slots. Each slot contains timers that should expire at the same time, organized as a doubly linked list.
2. `MinTmr_After` function selects a slot that a timer belongs into, and inserts the timer into that slot.
3. `MinSched_Trigger` function checks whether the timers of the next slot are expiring. If so, it invokes the callback function on each timer. `MinTmr_After` also implicitly invokes `MinSched_Trigger`.

The minute scheduler is faster than generic timer libraries because it does not maintain an ordered list of scheduled events, but simply puts them into a slot within an array.
It also consumes less memory because it records one callback function for all events instead of for each event.
It has the limitation of not being able to schedule events far in the future because there are a limited number of slots in the array.

The number of slots and the interval of each slot are specified in `MinSched_New` constructor. They affect how far in the future events could be scheduled. For example, setting 32 slots and 100ms interval allows scheduling at most 3100ms into the future.
