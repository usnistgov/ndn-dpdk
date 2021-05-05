import type { EalConfig, LCoreAllocConfig, PktmbufPoolTemplateUpdates } from "../dpdk";
import type { FwdpConfig } from "../fwdp";

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
export interface ActivateGenArgs {
  eal?: EalConfig;
  mempool?: PktmbufPoolTemplateUpdates<"DIRECT"|"INDIRECT"|"HEADER"|"INTEREST"|"DATA"|"PAYLOAD">;
  lcoreAlloc?: LCoreAllocConfig<"RX"|"TX"|"PRODUCER"|"CONSUMER">;

  /**
   * Minimum number of LCores to reserve.
   * Traffic generator on each face needs 3~5 LCores.
   * If there are fewer processor cores than LCores needed, use this option to create more LCores from threads.
   *
   * @TJS-type integer
   * @default 1
   */
  minLCores?: number;
}
