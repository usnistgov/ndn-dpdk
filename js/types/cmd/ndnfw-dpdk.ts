import type { LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FibConfig } from "../fib";
import type { CreateFaceConfig } from "../iface";
import type { NdtConfig } from "../ndt";
import type { SuppressConfig } from "../pit";
import type { PktQueueConfig } from "../pktqueue";

export interface FwdpInitConfig {
  FwdInterestQueue?: PktQueueConfig;
  FwdDataQueue?: PktQueueConfig;
  FwdNackQueue?: PktQueueConfig;
  LatencySampleFreq?: number;
  Suppress?: SuppressConfig;
  PcctCapacity?: number;
  CsCapMd?: number;
  CsCapMi?: number;
}

export interface NdnfwInitConfig {
  Mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"NAME"|"HEADER"|"GUIDER">;
  LCoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"CRYPTO"|"FWD">;
  Face?: CreateFaceConfig;
  Ndt?: NdtConfig;
  Fib?: FibConfig;
  Fwdp?: FwdpInitConfig;
}
