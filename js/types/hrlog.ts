import type { Counter } from "./core.js";
import type { LCore } from "./dpdk.js";

export enum HrlogAction {
  OI = 1,
  OD = 2,
  OC = 4,
}

export interface HrlogHistogram {
  Act: HrlogAction;
  LCore: LCore;
  Counts: Counter[];
}
