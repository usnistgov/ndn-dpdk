import type { Uint } from "./core";

type EalLCoreConfig = {
  cores?: number[];
  coresPerNuma?: QuantityPerNumaSocket;
  lcoresPerNuma?: QuantityPerNumaSocket;
  lcoreMain?: LCore;
} | {
  lcoreFlags?: string;
};

type EalMemoryConfig = {
  memChannels?: Uint;
  memPerNuma?: QuantityPerNumaSocket;
  filePrefix?: string;
  disableHugeUnlink?: boolean;
} | {
  memFlags?: string;
};

type EalDeviceConfig = {
  drivers?: string[];
  pciDevices?: string[];
  allPciDevices?: boolean;
  virtualDevices?: string[];
} | {
  deviceFlags?: string;
};

/**
 * Environment Abstraction Layer (EAL) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ealconfig#Config>
 */
export type EalConfig =
  (EalLCoreConfig & EalMemoryConfig & EalDeviceConfig & { extraFlags?: string }) |
  { flags?: string };

/**
 * NUMA socket number.
 * @minimum 0
 */
export type NumaSocket = Uint;

/**
 * @propertyNames { "pattern": "^\\d+$" }
 */
export type QuantityPerNumaSocket = Record<string, Uint>;

/**
 * DPDK logical core number.
 * @minimum 0
 */
export type LCore = Uint;

/**
 * LCore allocation configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ealthread#Config>
 */
export type LCoreAllocConfig<K extends string = string> = Partial<Record<K, LCoreAllocConfig.Role>>;

export namespace LCoreAllocConfig {
  /**
   * LCore allocation configuration for a role.
   * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ealthread#RoleConfig>
   */
  export type Role = LCore[] | QuantityPerNumaSocket;
}

/**
 * Packet mempool (template) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf#PoolConfig>
 */
export interface PktmbufPoolConfig {
  capacity?: Uint;
  privSize?: Uint;
  dataroom?: Uint;
}

/**
 * Packet mempool template updates.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf#TemplateUpdates>
 */
export type PktmbufPoolTemplateUpdates<K extends string = string> = Partial<Record<K, PktmbufPoolConfig>>;

/**
 * Preferences for creating virtual Ethernet device from kernel network interface.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethvdev#NetifConfig>
 */
export interface VDevNetifConfig {
  xdp?: {
    disabled?: boolean;
    args?: object;
    skipEthtool?: boolean;
  };
  afPacket?: {
    disabled?: boolean;
    args?: object;
  };
}
