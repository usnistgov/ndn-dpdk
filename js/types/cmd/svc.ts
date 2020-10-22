import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FwdpConfig } from "../fwdp";
import type { TgConfig } from "../tg/mod";

export interface ActivateFwArgs extends FwdpConfig {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"CRYPTO"|"FWD">;
}

export interface ActivateGenArgs extends TgConfig {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER"|"INTEREST"|"DATA"|"PAYLOAD">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"PRODUCER"|"CONSUMER">;
}
