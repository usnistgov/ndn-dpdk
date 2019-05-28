import { Counter } from "..";

export as namespace running_stat;

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
