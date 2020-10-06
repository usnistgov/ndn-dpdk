type EalLCoreConfig = {
  cores?: number[];
  coresPerNuma?: Record<number, number>;
} | {
  lcoreFlags?: string;
};

type EalMemoryConfig = {
  memChannels?: number;
  memPerNuma?: Record<number, number>;
} | {
  memFlags?: string;
};

type EalDeviceConfig = {
  pciDevices: string[];
  allPciDevices?: boolean;
  virtualDevices: string[];
} | {
  deviceFlags?: string;
};

export type EalConfig =
  (EalLCoreConfig & EalMemoryConfig & EalDeviceConfig & { extraFlags?: string }) |
  { flags?: string };

/**
 * DPDK logical core number.
 * @TJS-type integer
 * @minimum 0
 */
export type LCore = number;

export type LCoreAllocConfig<K extends string = string> = Record<K, LCoreAllocConfig.Role>;

export namespace LCoreAllocConfig {
  export interface Role {
    LCores?: LCore[];
    OnNuma?: Record<number, number>;
    EachNuma?: number;
  }
}

export interface PktmbufPoolConfig {
  Capacity: number;
  PrivSize: number;
  Dataroom: number;
}

export type PktmbufPoolTemplateUpdates<K extends string = string> = Record<K, PktmbufPoolConfig>;
