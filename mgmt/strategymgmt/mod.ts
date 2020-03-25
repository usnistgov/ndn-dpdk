import * as strategycode from "../../container/strategycode/mod";
import { Blob } from "../../core/mod";

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
  List: {args: {}; reply: StrategyInfo[]};
  Get: {args: IdArg; reply: StrategyInfo};
  Load: {args: LoadArg; reply: StrategyInfo};
  Unload: {args: IdArg; reply: StrategyInfo};
}
