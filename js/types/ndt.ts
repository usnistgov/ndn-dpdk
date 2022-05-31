import type { Uint } from "./core.js";

/**
 * Name Dispatch Table (NDT) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/ndt#Config>
 */
export interface NdtConfig {
  /**
   * @minimum 1
   * @maximum 17
   * @default 2
   */
  prefixLen?: Uint;

  /**
   * @minimum 16
   * @maximum 2147483648
   * @default 65536
   */
  capacity?: Uint;

  /**
   * @minimum 1
   * @maximum 1073741824
   * @default 1024
   */
  sampleInterval?: Uint;
}
