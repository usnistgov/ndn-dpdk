import { LCoreAllocConfig } from "../dpdk/mod";
import { Config as CreateFaceConfig } from "../iface/createface/mod";

export interface MempoolCapacityConfig {
  Capacity: number;
  DataroomSize?: number;
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
