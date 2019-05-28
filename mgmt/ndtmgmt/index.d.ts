import { Blob, Counter } from "../../core";
import * as ndn from "../../ndn";

export as namespace ndtmgmt;

interface UpdateArgBase {
  /**
   * @TJS-type integer
   * @minimum 0
   * @maximum 255
   */
  Value: number;
}

interface UpdateArgHash extends UpdateArgBase {
  /**
   * @TJS-type integer
   * @minimum 0
   */
  Hash: number;
}

interface UpdateArgName extends UpdateArgBase {
  Name: ndn.Name;
}

export type UpdateArgs = UpdateArgHash | UpdateArgName;

export interface UpdateReply {
  /**
   * @TJS-type integer
   */
  Index: number;
}

export interface NdtMgmt {
  ReadTable: {args: {}, reply: Blob};
  ReadCounters: {args: {}, reply: Counter[]};
  Update: {args: UpdateArgs, reply: UpdateReply};
}
