# ndn-dpdk/container/fib/fibtree

This package implements a tree structure used in [FIB](../).
It follows name hierarchy, where each edge is a name component and each node represents the name that joins all components from the root node.

The tree keeps track of depth of the subtree rooted at every node.
This is used as *MD* in FIB's 2-stage LPM algorithm.

The tree also maintains a list of nodes whose name length equals NDT prefix length, classified by NDT index of its name hash.
This is used for relocating affected FIB entries during an NDT update.
