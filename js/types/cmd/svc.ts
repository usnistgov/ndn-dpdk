import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FwdpConfig } from "../fwdp";

export interface ActivateArgsCommon {
  eal?: EalConfig;

  /** Whether to enable high resolution per-packet tracing module. */
  hrlog?: boolean;
}

/**
 * Forwarder activation arguments.
 * These are provided to the 'activate' mutation in GraphQL.
 */
export interface ActivateFwArgs extends ActivateArgsCommon, FwdpConfig {
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT" | "INDIRECT" | "HEADER">;
  lcoreAlloc?: LCoreAllocConfig<"RX" | "TX" | "CRYPTO" | "FWD">;
}

/**
 * Traffic generator activation arguments.
 * These are provided to the 'activate' mutation in GraphQL.
 */
export interface ActivateGenArgs extends ActivateArgsCommon {
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT" | "INDIRECT" | "HEADER" | "INTEREST" | "DATA" | "PAYLOAD">;
}
