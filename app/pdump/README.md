# ndn-dpdk/app/pdump

This package implements a packer dumper.
It collects NDN packets matching certain name prefixes as they enter or leave NDN-DPDK, and writes to a [pcapng](https://datatracker.ietf.org/doc/html/draft-tuexen-opsawg-pcapng) file.

## Writer

**Writer** type represents a packet dump writer thread that runs in an LCore of "PDUMP" role.
It receives packets from packet dump sources via a ring buffer, and writes them to a PCAP Next Generation (pcapng) file.

Three pcapng block types may appear in the output file:

* Section Header Block (SHB)
* Interface Description Block (IDB)
* Enhanced Packet Block (EPB)

SHB and IDB are prepared in Go code using [GoPacket library](https://pkg.go.dev/github.com/gopacket/gopacket/pcapgo), and then passed to C code via the ring buffer.
EPB is crafted directly in C code.

## Capturing from Face

**FaceSource** type defines a packet dump source attached to a face, on either incoming or outgoing direction.
It is referenced by **FaceImpl** in RCU protected pointers.
If assigned, every packet received or sent by a face is processed through `PdumpFace_Process` function.

The configuration contains one or more name prefixes under which the packet should be captured, and the probability of capturing a packet that matches the prefix.
It is possible to capture every packet by setting a `/` prefix with probability `1.0`.
Otherwise, the packet is parsed to extract its name, which is then compared to the list of prefixes.
If a packet is chosen to be captured, it is copied into a new mbuf, and sent to the **PdumpWriter**.

The packet parser for extracting name is greatly simplified compared to the [regular parser](../../ndni).
It understands both NDNLPv2 and NDN 0.3 packet format, but does not perform NDNLPv2 reassembly.
The parser can extract a portion of name that appears in the first fragment, but cannot process subsequent fragments.
The only way to capture non-first fragments is setting a `/` prefix as the only name filter entry, which disables the parser.

In the output file, each NDN-DPDK face appears as a separate network interface.
Packets are written as [Linux cooked-mode capture (SLL)](https://www.tcpdump.org/linktypes/LINKTYPE_LINUX_SLL.html) link type.
SLL is chosen instead of Ethernet because:

* By the time faceID is determined, the Ethernet header is already removed.
* In addition to Ethernet-based faces, NDN-DPDK also supports [socket faces](../../iface/socketface) where no Ethernet headers exist.

## Capturing from Ethernet Port

**EthPortSource** type defines a packet dump source attached to an [Ethernet port](../../iface/ethport), at a specific grab opportunity.
Currently, the only supported grab opportunity is *RxUnmatched*: it captures incoming packets on an Ethernet port that does not match any face.
It is referenced by **EthRxTable** table type in an RCU protected pointer.
Hence, this feature is only supported on Ethernet ports that use RxTable receive path.

In the output file, each Ethernet port appears as a network interface.
Packets are written as Ethernet link type, with the original Ethernet headers.
