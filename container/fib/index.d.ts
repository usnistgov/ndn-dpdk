import { Counter } from "../../core";

export as namespace fib;

export interface EntryCounters {
  NRxInterests: Counter;
  NRxData: Counter;
  NRxNacks: Counter;
  NTxInterests: Counter;
}
