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
  export interface PCI {
    driver: "PCI";
    pciAddr: string;
    devargs?: Record<string, string>;
  }

  export interface BifurcatedPCI {
    driver: "PCI";
    netif: string;
    devargs?: Record<string, string>;
  }

  export interface XDP {
    driver: "XDP";
    netif: string;
    devargs?: Record<string, string>;
    skipEthtool?: boolean;
  }

  export interface AfPacket {
    driver: "AF_PACKET";
    netif: string;
    devargs?: Record<string, string>;
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

  export type FileDriver = "aio" | "uring";

  export interface Nvme {
    pciAddr: string;
  }
}
