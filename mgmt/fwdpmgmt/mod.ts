import * as fwdp from "../../app/fwdp/mod";
import * as pit from "../../container/pit/mod";
import { Counter, Index } from "../../core/mod";

export interface IndexArg {
  Index: Index;
}

export interface FwdpInfo {
  NInputs: Counter;
  NFwds: Counter;
}

interface CsListCounters {
  Count: Counter;
  Capacity: Counter;
}

export interface CsCounters {
  MD: CsListCounters;
  MI: CsListCounters;
  NHits: Counter;
  NMisses: Counter;
}

export interface DpInfoMgmt {
  Global: {args: {}, reply: FwdpInfo};
  Input: {args: IndexArg, reply: fwdp.InputInfo};
  Fwd: {args: IndexArg, reply: fwdp.FwdInfo};
  Pit: {args: IndexArg, reply: pit.Counters};
  Cs: {args: IndexArg, reply: CsCounters};
}
