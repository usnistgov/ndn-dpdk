import type { Counter } from "./core";

export interface FibConfig {
  capacity?: number;
  nBuckets?: number;
  startDepth?: number;
}

export interface FibEntryCounters {
  NRxInterests: Counter;
  NRxData: Counter;
  NRxNacks: Counter;
  NTxInterests: Counter;
}
