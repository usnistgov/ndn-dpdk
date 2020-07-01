export interface HrlogFilenameArg {
  Filename: string;
}

export interface HrlogStartArgs extends HrlogFilenameArg {
  /**
   * @TJS-type integer
   * @default 268435456
   */
  Count?: number;
}

export interface HrlogMgmt {
  Start: {args: HrlogStartArgs; reply: {}};
  Stop: {args: HrlogFilenameArg; reply: {}};
}

export enum HrlogAction {
  OI = 1,
  OD = 2,
  OC = 4,
}
