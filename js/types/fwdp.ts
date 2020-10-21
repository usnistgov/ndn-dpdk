import type { FibConfig } from "./fib";
import type { NdtConfig } from "./ndt";
import type { PcctConfig } from "./pcct";
import type { SuppressConfig } from "./pit";
import type { PktQueueConfig } from "./pktqueue";

export interface FwdpConfig {
  ndt?: NdtConfig;
  fib?: FibConfig;
  pcct?: PcctConfig;
  suppress?: SuppressConfig;
  crypto?: FwdpCryptoConfig;
  fwdInterestQueue?: PktQueueConfig;
  fwdDataQueue?: PktQueueConfig;
  fwdNackQueue?: PktQueueConfig;
  latencySampleFreq?: number;
}

export interface FwdpCryptoConfig {
  inputCapacity?: number;
  opPoolCapacity?: number;
}
