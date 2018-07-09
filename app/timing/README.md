# ndn-dpdk/app/timing

This package implements a per-packet latency timing tracer.
The [forwarding data plane](../fwdp/) measures and posts timing entries to a ring buffer, and a timing writer saves these entries to a binary log file.

## Timed Actions

`enum TimingAction` declares what data plane actions being timed.

From FwInput, **IN** dispatches Interest packets by name, **IT** dispatches Data/Nack packets by token.
These timings do not include decoding, which happened prior to FwInput despite being executed in the same lcore.

From FwFwd, **FI**, **FD**, and **FN** process Interest, Data, or Nack packets.
These actions do not dispatch the outcome of packet processing.
For example, **FD** can refer to either a Data matching a PIT entry and returned to downstream, or an unsolicited Data dropped.

## Activation

[ndnfw-dpdk](../../cmd/ndnfw-dpdk/) program activates this timing tracer when `TIMING_WRITER` environment variable is present.

`TIMING_WRITER` environment variable has four fields, separated by colon:

1.  RingCapacity: capacity of ring buffer.
    Entries will be lost if the ring buffer is full.
2.  NTotal: how many entries to collect.
    The timing writer (and ndnfw-dpdk program) terminates after collecting enough entries.
    Lost and discarded entries do not count toward NTotal.
3.  NSkip: how many initial entries to discard, in order to skip warm-up stage.
4.  Filename: where to write log file.

## Log File Format

The log file starts with a 16-byte header, followed by 8-byte entries.

The header has three fields:

1.  32-bit magic number 0x35f0498a, written in native endianness.
    If this number appears endian-swapped, the reader shall swap all other multi-byte numbers.
2.  32-bit version number.
    Currently it is 1.
3.  64-bit TSCHZ.
    This is "1 second" represented in TSC duration unit.

Each entry has three fields:

1.  8-bit action type. See `entry.h`.
2.  8-bit lcore id.
3.  48-bit duration.
