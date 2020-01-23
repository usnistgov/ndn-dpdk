import { Counter } from "../../core/mod.js";

export interface EntryCounters {
  NRxInterests: Counter;
  NRxData: Counter;
  NRxNacks: Counter;
  NTxInterests: Counter;
}
