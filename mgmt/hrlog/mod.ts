export interface FilenameArg {
  Filename: string;
}

export interface StartArgs extends FilenameArg {
  /**
   * @TJS-type integer
   * @default 268435456
   */
  Count?: number;
}

export interface HrlogMgmt {
  Start: {args: StartArgs; reply: {}};
  Stop: {args: FilenameArg; reply: {}};
}

export enum HrlogAction {
  OI = 1,
  OD = 2,
  OC = 4,
}
