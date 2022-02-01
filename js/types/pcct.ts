import type { Uint } from "./core";

/**
 * PIT-CS Composite Table (PCCT) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/pcct#Config>
 */
export interface PcctConfig {
  pcctCapacity?: Uint;
  csMemoryCapacity?: Uint;
  csDiskCapacity?: Uint;
  csIndirectCapacity?: Uint;
}
