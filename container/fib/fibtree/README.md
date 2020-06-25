# ndn-dpdk/container/fib/fibtree

This package implements the tree structure used by the [FIB](../).
It follows the name hierarchy, where each edge is a name component and each node represents the name that is formed by concatenating all components starting from the root node.

The tree keeps track of the depth of the subtree rooted at every node.
This is used as *MD* in the FIB's 2-stage LPM algorithm.

The tree also maintains a list of nodes whose name length equals the NDT prefix length, classified by NDT index of its name hash.
This is used for relocating the affected FIB entries during an NDT update.
