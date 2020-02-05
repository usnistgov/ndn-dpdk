import { Nanoseconds } from "../../core/nnduration/mod.js";

export interface Config {
  /**
   * @TJS-type integer
   * @minimum 64
   */
  Capacity: number;

  /**
   * @TJS-type integer
   * @default 64
   * @minimum 1
   * @maximum 64
   */
  DequeueBurstSize: number;

  /**
   * @default 0
   */
  Delay: Nanoseconds;

  DisableCoDel: boolean;

  /**
   * @default 5000000
   */
  Target: Nanoseconds;

  /**
   * @default 100000000
   */
  Interval: Nanoseconds;
}
