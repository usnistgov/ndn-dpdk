# ndn-dpdk/cmd/nfdemu

This directory contains a NFD emulator that allows application written for NFD to work with ndn-dpdk forwarder.

The emulator listens on a Unix stream socket.
When a client application is connected, the emulator creates a face on the forwarder, transfers Interest/Data/Nack between the client and the forwarder, and automatically inserts and strips the PIT token element as necessary.
The emulator recognizes NFD-style prefix registration commands and translates them into FIB update commands.

## Usage

Before starting nfdemu, launch [ndnfw-dpdk](../ndnfw-dpdk/) and [mgmtproxy.sh](../mgmtclient/)/

Start nfdemu:

    nodejs build/cmd/nfdemu/

Run NDN producer program:

    NDN_CLIENT_TRANSPORT=unix:///tmp/nfdemu.sock ndnpingserver /Z

Run NDN consumer program:

    NDN_CLIENT_TRANSPORT=unix:///tmp/nfdemu.sock ndnping /Z
