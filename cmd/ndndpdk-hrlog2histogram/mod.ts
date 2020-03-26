import { HrlogAction } from "../../mgmt/hrlog/mod";

export interface Histogram {
  Act: HrlogAction;
  LCore: number;
  Counts: number[];
}
