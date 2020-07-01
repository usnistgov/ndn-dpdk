import type { Blob, Index } from "../core";
import type { IdArg } from "./common";

export interface StrategyMgmt {
  List: {args: {}; reply: StrategyInfo[]};
  Get: {args: IdArg; reply: StrategyInfo};
  Load: {args: StrategyLoadArg; reply: StrategyInfo};
  Unload: {args: IdArg; reply: StrategyInfo};
}

export interface StrategyInfo {
  Id: Index;
  Name: string;
}

export interface StrategyLoadArg {
  Name: string;
  Elf: Blob;
}
