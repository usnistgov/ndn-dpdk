import type { LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";

export interface NdnpingInitConfig {
  Mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER"|"INTEREST"|"DATA"|"PAYLOAD">;
  LCoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"SVR"|"CLIR"|"CLIT">;
}
