import { Nanoseconds } from "../../core/nnduration/mod.js";

export interface Config {
  /**
   * @default 5000000
   */
  Target: Nanoseconds;

  /**
   * @default 100000000
   */
  Interval: Nanoseconds;

  /**
   * @TJS-type integer
   * @default 64
   * @minimum 1
   * @maximum 64
   */
  DequeueBurstSize: number;
}
