type EalLCoreConfig = {
  cores?: number[];
  coresPerNuma?: Record<number, number>;
} | {
  lcoreFlags?: string;
};

type EalMemoryConfig = {
  memChannels?: number;
  memPerNuma?: Record<number, number>;
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
 * DPDK logical core number.
 * @TJS-type integer
 * @minimum 0
 */
export type LCore = number;

/**
 * LCore allocation configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ealthread#AllocConfig>
 */
export type LCoreAllocConfig<K extends string = string> = Partial<Record<K, LCoreAllocConfig.Role>>;

export namespace LCoreAllocConfig {
  /**
   * LCore allocation configuration for a role.
   * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ealthread#AllocRoleConfig>
   */
  export interface Role {
    lcores?: LCore[];
    onNuma?: Record<number, number>;
    eachNuma?: number;
  }
}

/**
 * Packet mempool (template) configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf#PoolConfig>
 */
export interface PktmbufPoolConfig {
  capacity?: number;
  privSize?: number;
  dataroom?: number;
}

/**
 * Packet mempool template updates.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf#TemplateUpdates>
 */
export type PktmbufPoolTemplateUpdates<K extends string = string> = Partial<Record<K, PktmbufPoolConfig>>;
