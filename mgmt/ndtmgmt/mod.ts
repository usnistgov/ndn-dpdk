import { Blob, Counter } from "../../core/mod";
import * as ndni from "../../ndni/mod";

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
  Name: ndni.Name;
}

export type UpdateArgs = UpdateArgHash | UpdateArgName;

export interface UpdateReply {
  /**
   * @TJS-type integer
   */
  Index: number;
}

export interface NdtMgmt {
  ReadTable: {args: {}; reply: Blob};
  ReadCounters: {args: {}; reply: Counter[]};
  Update: {args: UpdateArgs; reply: UpdateReply};
}
