# ndn-dpdk/app/ndnping

This package implements a NDN packet generator that doubles as **ndnping** client and server.

Unlike named-data.net's [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) and [ndn-traffic-generator](https://github.com/named-data/ndn-traffic-generator), this implementation does not use a local forwarder, but directly sends and receives packets on a network interface.

This packet generator has five kinds of threads:

*   A single **input thread** ("RX" role) runs an *iface.RxLoop* that invokes `NdnpingInput_FaceRx` when a burst of L3 packets arrives on any face.
    It dispatches Data and Nacks to client-RX threads, and dispatches Interests to server threads.
*   A per-face **client-TX thread** ("CLIT" role) executes `NdnpingClient_RunTx` function that periodically sends Interests.
*   A per-face **client-RX thread** ("CLIR" role) executes `NdnpingClient_RunRx` function that receives Data and Nacks from the input thread, and collects statistics about them.
*   A per-face **server thread** ("SVR" role) executes `NdnpingServer_Run` function that receives Interest from the input thread via a queue, and responds to them.
*   A single **output thread** ("TX" role) runs an *iface.TxLoop* that transmits Interests, Data, and Nacks created by any client-RX thread or server thread.

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

The `Client` sends Interests and receives Data or Nacks.
It supports multiple *patterns* i.e. name prefixes.
The name suffix is a 64-bit sequence number encoded in binary format and in native endianness.
Other aspects of the Interest, such as InterestLifetime, are currently hard-coded.

Strictly speaking, binary sequence numbers violate the [ndnping Protocol](https://github.com/named-data/ndn-tools/blob/1fda67dc75692ccf0283a410f70db55686e2ff48/tools/ping/README.md#ndnping-protocol) that requires the sequence number to be encoded as ASCII.
However, the current C++ `ndnpingserver` implementation can respond to such Interests.

The client maintains Interest, Data, Nack counters under each pattern.
It also writes Interest sending time in the "PIT token" field, and uses that to collect round-trip time of Data retrievals.

## Server

The `Server` responds to every Interest with Data or Nack.
It supports multiple *patterns* i.e. name prefixes.
Interests that fall under one of the prefixes are responded with Data.
Optionally, the server can respond Nack to Interests not matching any pattern.

The server maintains counters for the number of processed Interests under each pattern.
