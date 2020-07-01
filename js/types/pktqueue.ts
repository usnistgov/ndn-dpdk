import type { NNNanoseconds } from "./core";

export type PktQueueConfig = PktQueueConfig.Plain | PktQueueConfig.Delay | PktQueueConfig.CoDel;

export namespace PktQueueConfig {
  interface Common {
    /**
     * @TJS-type integer
     * @minimum 64
     */
    Capacity?: number;

    /**
     * @TJS-type integer
     * @default 64
     * @minimum 1
     * @maximum 64
     */
    DequeueBurstSize?: number;
  }

  export interface Plain extends Common {
    DisableCoDel: true;
  }

  export interface Delay extends Common {
    Delay: NNNanoseconds;
  }

  export interface CoDel extends Common {
    DisableCoDel?: false;

    /**
     * @default 5000000
     */
    Target?: NNNanoseconds;

    /**
     * @default 100000000
     */
    Interval?: NNNanoseconds;
  }
}
