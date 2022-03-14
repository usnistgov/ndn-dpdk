import type { Uint } from "./core";

/**
 * PIT-CS Composite Table (PCCT) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/pcct#Config>
 */
export interface PcctConfig {
  /**
   * @minimum 64
   * @maximum 2147483647
   * @default 131071
   */
  pcctCapacity?: Uint;

  /**
   * @minimum 64
   * @maximum 2147483647
   */
  csMemoryCapacity?: Uint;

  /**
   * @minimum 0
   * @maximum 2147483647
   * @default 0
   */
  csDiskCapacity?: Uint;

  /**
   * @minimum 64
   * @maximum 2147483647
   */
  csIndirectCapacity?: Uint;
}
