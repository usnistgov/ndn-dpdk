import { Counter } from "../../core/mod";
import { Nanoseconds } from "../../core/nnduration/mod";

export interface SuppressConfig {
  /**
   * @minimum 0
   * @default 10E6
   */
  Min?: Nanoseconds;

  /**
   * @minimum 0
   * @default 100E6
   */
  Max?: Nanoseconds;

  /**
   * @minimum 1.0
   * @default 2.0
   */
  Multiplier?: number;
}

export interface Counters {
  NEntries: Counter;
  NInsert: Counter;
  NFound: Counter;
  NCsMatch: Counter;
  NAllocErr: Counter;
  NDataHit: Counter;
  NDataMiss: Counter;
  NNackHit: Counter;
  NNackMiss: Counter;
  NExpired: Counter;
}
