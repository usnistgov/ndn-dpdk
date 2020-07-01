import type { Blob, Counter, Name } from "../core";

export interface NdtMgmt {
  ReadTable: {args: {}; reply: Blob};
  ReadCounters: {args: {}; reply: Counter[]};
  Update: {args: NdtUpdateArgs; reply: NdtUpdateReply};
}

export type NdtUpdateArgs = NdtUpdateArgs.ByHash | NdtUpdateArgs.ByName;

export namespace NdtUpdateArgs {
  interface Common {
    /**
     * @TJS-type integer
     * @minimum 0
     * @maximum 255
     */
    Value: number;
  }

  export interface ByHash extends Common {
    /**
     * @TJS-type integer
     * @minimum 0
     */
    Hash: number;
  }

  export interface ByName extends Common {
    Name: Name;
  }
}

export interface NdtUpdateReply {
  /**
   * @TJS-type integer
   */
  Index: number;
}
