import * as strategycode from "../../container/strategycode";
import { Blob } from "../../core";

export as namespace strategymgmt;

export interface IdArg {
  Id: strategycode.Id;
}

export interface StrategyInfo {
  Id: strategycode.Id;
  Name: string;
}

export interface LoadArg {
  Name: string;
  Elf: Blob;
}

export interface StrategyMgmt {
  List: {args: {}, reply: StrategyInfo[]};
  Get: {args: IdArg, reply: StrategyInfo};
  Load: {args: LoadArg, reply: StrategyInfo};
  Unload: {args: IdArg, reply: StrategyInfo};
}
