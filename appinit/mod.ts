import { LCoreAllocConfig } from "../dpdk/mod.js";
import { Config as CreateFaceConfig } from "../iface/createface/mod.js";

export interface MempoolCapacityConfig {
  Capacity: number;
  CacheSize?: number;
  DataroomSize?: number;
}

export namespace MempoolCapacityConfig {
  export function create(capacity: number, dataroomSize?: number) {
    const cfg = {
      Capacity: capacity,
    } as MempoolCapacityConfig;
    if (dataroomSize) {
      cfg.DataroomSize = dataroomSize;
    }
    setCacheSize(cfg);
    return cfg;
  }

  function setCacheSize(cfg: MempoolCapacityConfig) {
    const { Capacity: capacity } = cfg;
    cfg.CacheSize = 512;
    for (let cacheSize = 512; cacheSize >= 64; --cacheSize) {
      if (capacity % cacheSize === 0) {
        cfg.CacheSize = cacheSize;
      }
    }
  }
}

export interface MempoolsCapacityConfig {
  IND?: MempoolCapacityConfig;
  ETHRX?: MempoolCapacityConfig;
  NAME?: MempoolCapacityConfig;
  HDR?: MempoolCapacityConfig;
  INTG?: MempoolCapacityConfig;
  INT?: MempoolCapacityConfig;
  DATA0?: MempoolCapacityConfig;
  DATA1?: MempoolCapacityConfig;
}

export interface InitConfig {
  Mempool?: MempoolsCapacityConfig;
  LCoreAlloc?: LCoreAllocConfig;
  Face?: CreateFaceConfig;
}
