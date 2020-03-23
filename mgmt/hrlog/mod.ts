export interface CollectArgs {
  Filename: string;

  /**
   * @TJS-type integer
   */
  Count: number;
}

export interface HrlogMgmt {
  Collect: {args: CollectArgs, reply: {}};
}
