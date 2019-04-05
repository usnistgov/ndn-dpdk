# ndn-dpdk/cmd/nfdemu

This directory contains a NFD emulator that allows application written for NFD to work with ndn-dpdk forwarder.

The emulator listens on a Unix stream socket.
When a client application is connected, the emulator creates a face on the forwarder, transfers Interest/Data/Nack between the client and the forwarder, and automatically inserts and strips the PIT token element as necessary.
The emulator recognizes NFD-style prefix registration commands and translates them into FIB update commands.

## Usage

In `$HOME/.ndn/client.conf`, write:

    transport=unix:///tmp/nfdemu.sock

Start ndnfw-dpdk:

    sudo MGMT=tcp4://127.0.0.1:6345 ndnfw-dpdk

Run NDN producer program:

    NDN_CLIENT_TRANSPORT=unix:///tmp/nfdemu.sock ndnpingserver /Z

Run NDN consumer program:

    NDN_CLIENT_TRANSPORT=unix:///tmp/nfdemu.sock ndnping -a /Z

## Limitations

NDN.JS v0.16 only accepts NDN Packet Format v0.2, while ndn-dpdk only accepts NDN Packet Format v0.3.
As a result, a packet can go through only if it is valid under both v0.2 and v0.3 formats.
