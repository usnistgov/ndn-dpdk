import type { Uint } from "./core.js";
import type { BdevLocator } from "./dpdk.js";
import type { FibConfig } from "./fib.js";
import type { NdtConfig } from "./ndt.js";
import type { PcctConfig } from "./pcct.js";
import type { SuppressConfig } from "./pit.js";
import type { PktQueueConfig } from "./pktqueue.js";

/**
 * Forwarder data plane configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/fwdp#Config>
 */
export interface FwdpConfig {
  ndt?: NdtConfig;
  fib?: FibConfig;
  pcct?: PcctConfig;
  suppress?: SuppressConfig;
  crypto?: FwdpCryptoConfig;
  disk?: FwdpDiskConfig;
  fwdInterestQueue?: PktQueueConfig;
  fwdDataQueue?: PktQueueConfig;
  fwdNackQueue?: PktQueueConfig;
  latencySampleInterval?: Uint;
}

export interface FwdpCryptoConfig {
  inputCapacity?: Uint;
  opPoolCapacity?: Uint;
}

export type FwdpDiskConfig = BdevLocator & {
  /**
   * @min 1.00
   * @default 1.05
   */
  overprovision?: number;
};
