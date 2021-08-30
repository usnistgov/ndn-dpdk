import type { Uint } from "./core";

/**
 * Name Dispatch Table (NDT) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/ndt#Config>
 */
export interface NdtConfig {
  prefixLen?: Uint;
  capacity?: Uint;
  sampleInterval?: Uint;
}
