# ndn-dpdk/app/pingserver

This package is part of the [packet generator](../ping/).
It implements an **ndnping** server.
It runs `PingServer_Run` function in a *server thread* ("SVR" role) of the traffic generator.

The server responds to every Interest with Data or Nack.
It supports multiple patterns that allow setting:

* Name prefix
* a list of possible reply definitions, each with a probability of selecting this reply relative to other replies, and one of:
  * Data template: Name suffix, FreshnessPeriod value, Content payload length
  * Nack reason
  * timeout/drop

Upon receiving an Interest, the server finds a pattern whose name prefix is a prefix of Interest name, and randomly selects a reply definition.
It then makes a Data or Nack according to the reply definition, unless it's a "timeout" reply.
In case of a Data reply, the Data Name is the Interest name combined with the configured name suffix; if name suffix is non-empty, the Interest needs to set CanBePrefix.
If no pattern matches the Interest, the server can optionally respond a Nack.

The server maintains counters for the number of processed Interests under each pattern and reply definition, and a counter for non-matching Interests.

The server can enforce a minimum processing delay.
The response to each packet would not leave the server thread until the minimum processing delay has elapsed since the packet arrives at the input thread.
For efficiency, this delay is applied per burst, using the timestamp on the last packet of the burst.
