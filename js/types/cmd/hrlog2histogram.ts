import type { HrlogAction } from "../mgmt/hrlog";

export interface Histogram {
  Act: HrlogAction;
  LCore: number;
  Counts: number[];
}
