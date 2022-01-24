import type { Counter } from "./core";
import type { LCore } from "./dpdk";

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
