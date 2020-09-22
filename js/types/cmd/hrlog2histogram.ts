export enum HrlogAction {
  OI = 1,
  OD = 2,
  OC = 4,
}

export interface Histogram {
  Act: HrlogAction;
  LCore: number;
  Counts: number[];
}
