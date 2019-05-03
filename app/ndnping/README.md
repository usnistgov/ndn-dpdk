# ndn-dpdk/app/ndnping

This package implements a NDN packet generator that doubles as **ndnping** client and server.

Unlike named-data.net's [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) and [ndn-traffic-generator](https://github.com/named-data/ndn-traffic-generator), this implementation does not use a local forwarder, but directly sends and receives packets on a network interface.

This packet generator has up to five threads for each face:

*   The **input thread** ("RX" role) runs an *iface.RxLoop* that invokes `PingInput_FaceRx` when a burst of L3 packets arrives on a face.
    It dispatches Data and Nacks to client-RX thread, and dispatches Interests to server thread.
*   The **client-TX thread** ("CLIT" role) executes `PingClientTx_Run` function that periodically sends Interests.
*   The **client-RX thread** ("CLIR" role) executes `PingClientRx_Run` function that receives Data and Nacks from the input thread, and collects statistics about them.
*   The **server thread** ("SVR" role) executes `PingServer_Run` function that receives Interests from the input thread, and responds to them.
*   The **output thread** ("TX" role) runs an *iface.TxLoop* that transmits Interests, Data, and Nacks created by client-RX thread or server thread.

```
      /--client0-RX
      |             client0-TX--\
      |                         |
input-+--client1-RX             |
      |             client1-TX--+-output
      |                         |
      +---------server0---------+
      \---------server1---------/
```

## Client

The client sends Interests and receives Data or Nacks.
It supports multiple patterns that allow setting:

* probability weight of selecting this pattern relative to other patterns
* Name prefix
* CanBePrefix flag
* MustBeFresh flag
* InterestLifetime value
* HopLimit value

The client randomly selects a pattern, and makes an Interest with the pattern settings.
The Interest name ends with a sequence number, which is a 64-bit number encoded in binary format and native endianness.
Strictly speaking, these sequence numbers violate the [ndnping Protocol](https://github.com/named-data/ndn-tools/blob/1fda67dc75692ccf0283a410f70db55686e2ff48/tools/ping/README.md#ndnping-protocol) that requires the sequence number to be encoded as ASCII.
However, the current C++ `ndnpingserver` implementation can respond to such Interests.

The client maintains Interest, Data, Nack counters and collects Data round-trip time under each pattern.

### PIT token usage

The client encodes these information in the PIT token field:

* when was the Interest sent
* which pattern created the Interest
* a "run number" so that replies to Interests from previous executions would not be accepted

Having these with the packet eliminates the need for a pending Interest table, allowing the client to operate more efficiently.
However, the client would not be able to detect network faults, such as unsolicited replies and double replies.

## Server

The server responds to every Interest with Data or Nack.
It supports multiple patterns that allow setting:

* Name prefix
* Name suffix
* FreshnessPeriod value
* Content payload length

Upon receiving an Interest, the server finds a pattern whose name prefix is a prefix of Interest name, and makes a Data with the pattern settings.
The Data name is the Interest name combined with the name suffix configured in the pattern; note that if name suffix is non-empty, the Interest needs to set CanBePrefix.
If no pattern matches the Interest, the server can optionally respond a Nack.

The server maintains counters for the number of processed Interests under each pattern, and a counter for non-matching Interests.
