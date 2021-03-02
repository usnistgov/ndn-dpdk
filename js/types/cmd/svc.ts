import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FwdpConfig } from "../fwdp";
import type { TgConfig } from "../tg/mod";

/**
 * Forwarder activation arguments.
 * These are provided to the 'activate' mutation in GraphQL.
 */
export interface ActivateFwArgs extends FwdpConfig {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"CRYPTO"|"FWD">;
}

/**
 * Traffic generator activation arguments.
 * These are provided to the 'activate' mutation in GraphQL.
 */
export interface ActivateGenArgs extends TgConfig {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER"|"INTEREST"|"DATA"|"PAYLOAD">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"PRODUCER"|"CONSUMER">;
}
