import * as fwdp from "../../app/fwdp/mod.js";
import * as pit from "../../container/pit/mod.js";
import { Counter, Index } from "../../core/mod.js";

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

export interface FwdpMgmt {
  Global: {args: {}, reply: FwdpInfo};
  Input: {args: IndexArg, reply: fwdp.InputInfo};
  Fwd: {args: IndexArg, reply: fwdp.FwdInfo};
  Pit: {args: IndexArg, reply: pit.Counters};
  Cs: {args: IndexArg, reply: CsCounters};
}
