# ndn-dpdk/container/hrlog

This package implements a high resolution logger, suitable for per-packet tracing.

## Activation

User should invoke `Start` function or GraphQL `collectHrlog` mutation to start collecting log entries to a file, and invoke `Stop` function or GraphQL `delete` mutation to stop.
Only one collection can run at any moment.
Log entries posted when collection is not running are lost.

## Log File Format

The log file starts with a 16-byte header, followed by 8-byte entries.

The header has three fields:

1. 32-bit magic number 0x35f0498a, written in native endianness.
   If this number appears endian-swapped, the reader shall swap all other multi-byte numbers.
2. 32-bit version number. See `entry.h`.
   This is incremented whenever action types have a backwards incompatible change.
3. 64-bit TSCHZ.
   This is "1 second" represented in TSC duration unit.

Each entry has three fields:

1. 48-bit value.
   If this is a duration, it is in TSC unit.
2. 8-bit lcore id.
3. 8-bit action type. See `entry.h`.

## Integration

To integrate this package in NDN-DPDK codebase:

1. Include `entry.h` header where log entries are generated, and invoke functions in that header.
2. Invoke Go `Init` function after EAL initialization.
