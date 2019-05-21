# ndn-dpdk/iface/createface

This package implements face creation procedures.
It offers a `Create` function that creates a face from an **iface.Locator**.

Before invoking `Create`, the caller must initialize this package:

1. Construct and `Apply` a **Config** that contains static configuration options.
2. Provide mempools via `AddMempool` function.
3. Provide RxLoops and TxLoops via `AddRxLoop` and `AddTxLoop` functions.

Ideally, NUMA placement of mempools, RxLoops, and TxLoops should follow existing and anticipated faces.
`ListRxTxNumaSockets` function (available after `Config.Apply`) can recommend where to place these resources, according to enabled face kinds and probed PCI devices.

`Create` would work as long as at least one mempool set, one RxLoop, and one TxLoop have been added.
When multiple are available, those on the same NUMA socket are preferred, and RxLoop/TxLoop serving fewer RxGroups and Faces are preferred.
