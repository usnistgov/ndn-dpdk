import type { Counter } from "./core";

export interface FibConfig {
  MaxEntries: number;
  NBuckets: number;
  StartDepth: number;
}

export interface FibEntryCounters {
  NRxInterests: Counter;
  NRxData: Counter;
  NRxNacks: Counter;
  NTxInterests: Counter;
}
