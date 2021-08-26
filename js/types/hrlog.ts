export interface HrlogWriterConfig {
  ringCapacity?: number;
}

export enum HrlogAction {
  OI = 1,
  OD = 2,
  OC = 4,
}

export interface HrlogHistogram {
  Act: HrlogAction;
  LCore: number;
  Counts: number[];
}
