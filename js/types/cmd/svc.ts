import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FwdpConfig } from "../fwdp";

export interface ActivateArgsCommon<Roles extends string = never> {
  eal?: EalConfig;

  lcoreAlloc?: LCoreAllocConfig<Roles | "HRLOG" | "PDUMP">;
}

/**
 * Forwarder activation arguments.
 * These are provided to the 'activate' mutation in GraphQL.
 */
export interface ActivateFwArgs extends ActivateArgsCommon<"RX" | "TX" | "CRYPTO" | "FWD">, FwdpConfig {
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT" | "INDIRECT" | "HEADER">;
}

/**
 * Traffic generator activation arguments.
 * These are provided to the 'activate' mutation in GraphQL.
 */
export interface ActivateGenArgs extends ActivateArgsCommon {
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT" | "INDIRECT" | "HEADER" | "INTEREST" | "DATA" | "PAYLOAD">;
}
