import type { Counter } from "../core";
import type { FwdpFwdInfo, FwdpInputInfo } from "../fwdp";
import type { PitCounters } from "../pit";
import type { IndexArg } from "./common";

export interface DpInfoMgmt {
  Global: {args: {}; reply: FwdpInfo};
  Input: {args: IndexArg; reply: FwdpInputInfo};
  Fwd: {args: IndexArg; reply: FwdpFwdInfo};
  Pit: {args: IndexArg; reply: PitCounters};
  Cs: {args: IndexArg; reply: CsCounters};
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
