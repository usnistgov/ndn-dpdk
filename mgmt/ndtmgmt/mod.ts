import { Blob, Counter } from "../../core/mod.js";
import * as ndn from "../../ndn/mod.js";

interface UpdateArgCommon {
  /**
   * @TJS-type integer
   * @minimum 0
   * @maximum 255
   */
  Value: number;
}

interface UpdateArgHash extends UpdateArgCommon {
  /**
   * @TJS-type integer
   * @minimum 0
   */
  Hash: number;
}

interface UpdateArgName extends UpdateArgCommon {
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
