import type { Uint } from "./core.js";

/**
 * Forwarding Information Base (FIB) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/fib/fibdef#Config>
 */
export interface FibConfig {
  capacity?: Uint;
  nBuckets?: Uint;
  startDepth?: Uint;
}
