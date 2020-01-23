import { Counter } from "../mod.js";

export interface Snapshot {
  Count: Counter;
  Min: number;
  Max: number;
  Mean: number;

  /**
   * @minimum 0
   */
  Stdev: number;
}
