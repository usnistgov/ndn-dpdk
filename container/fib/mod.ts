import { Counter } from "../../core/mod.js";

export interface Config {
	Id: string;
	MaxEntries: number;
	NBuckets: number;
	StartDepth: number;
}

export type ConfigTemplate = Partial<Omit<Config, "Id">>;

export interface EntryCounters {
  NRxInterests: Counter;
  NRxData: Counter;
  NRxNacks: Counter;
  NTxInterests: Counter;
}
