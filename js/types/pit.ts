import type { NNNanoseconds } from "./core";

export interface SuppressConfig {
  /**
   * @minimum 0
   * @default 10E6
   */
  min?: NNNanoseconds;

  /**
   * @minimum 0
   * @default 100E6
   */
  max?: NNNanoseconds;

  /**
   * @minimum 1.0
   * @default 2.0
   */
  multiplier?: number;
}
