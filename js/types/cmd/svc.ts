import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FwdpConfig } from "../fwdp";
import type { PingConfig } from "../ping/mod";

export interface ActivateFwArgs extends FwdpConfig {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"CRYPTO"|"FWD">;
}

export interface ActivateGenArgs extends PingConfig {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER"|"INTEREST"|"DATA"|"PAYLOAD">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"SVR"|"CLIR"|"CLIT">;
}
