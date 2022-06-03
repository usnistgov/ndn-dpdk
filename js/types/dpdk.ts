import type { Uint } from "./core.js";

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
  preallocateMem?: boolean;
  filePrefix?: string;
} | {
  memFlags?: string;
};

type EalDeviceConfig = {
  iovaMode?: "PA" | "VA";
  drivers?: string[];
  disablePCI?: boolean;
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
 * EthDev selection and creation arguments.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif#Config>
 */
export type EthNetifConfig = EthNetifConfig.PCI | EthNetifConfig.BifurcatedPCI | EthNetifConfig.XDP | EthNetifConfig.AfPacket;

export namespace EthNetifConfig {
  interface Base {
    devargs?: Record<string, string>;
  }

  export interface PCI extends Base {
    driver: "PCI";
    pciAddr: string;
  }

  export interface BifurcatedPCI extends Base {
    driver: "PCI";
    netif: string;
  }

  export interface XDP extends Base {
    driver: "XDP";
    netif: string;
    skipEthtool?: boolean;
  }

  export interface AfPacket extends Base {
    driver: "AF_PACKET";
    netif: string;
  }
}

/**
 * SPDK block device creation parameters.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/dpdk/bdev#Locator>
 */
export type BdevLocator = BdevLocator.Malloc | BdevLocator.File | BdevLocator.Nvme;

export namespace BdevLocator {
  export interface Malloc {
    malloc: true;
  }

  export interface File {
    file: string;
    fileDriver?: FileDriver;
  }

  export type FileDriver = "aio" | "uring.js";

  export interface Nvme {
    pciAddr: string;
  }
}
