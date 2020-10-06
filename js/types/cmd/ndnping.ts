import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";

export interface NdnpingInitConfig {
  eal?: EalConfig;
  Mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER"|"INTEREST"|"DATA"|"PAYLOAD">;
  LCoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"SVR"|"CLIR"|"CLIT">;
}
