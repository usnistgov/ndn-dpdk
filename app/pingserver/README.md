# ndn-dpdk/app/pingserver

This package is part of the [packet generator](../ping).
It implements an **ndnping** server.
It runs the `PingServer_Run` function in a *server thread* ("SVR" role) of the traffic generator.

The server responds to every Interest with Data or Nack.
It supports multiple configurable patterns:

* Name prefix
* a list of possible response definitions, each with a relative probability of being selected and one of:
  * Data template: Name suffix, FreshnessPeriod value, Content payload length
  * Nack reason
  * timeout/drop

Upon receiving an Interest, the server finds a pattern whose name prefix is a prefix of the Interest name, and randomly picks a response definition.
It then creates a Data or Nack packet according to the selected definition, unless the latter specifies a "timeout".
In case of a Data reply, the Data Name is the Interest name combined with the configured name suffix; if the name suffix is non-empty, the Interest needs to have the CanBePrefix flag.
If no pattern matches the Interest, the server can optionally reply with a Nack.

The server maintains counters for the number of processed Interests under each pattern and response definition, and a counter for non-matching Interests.

The server can emulate a minimum processing delay for testing purposes.
If this is enabled, the reply to each packet will not leave the server thread until the minimum processing delay has elapsed since the packet arrived at the input thread.
For efficiency reasons, this delay is applied per burst of packets, using the timestamp on the last packet of each burst.
