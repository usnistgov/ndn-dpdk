import type { Counter, NNNanoseconds } from "./core";

export interface SuppressConfig {
  /**
   * @minimum 0
   * @default 10E6
   */
  Min?: NNNanoseconds;

  /**
   * @minimum 0
   * @default 100E6
   */
  Max?: NNNanoseconds;

  /**
   * @minimum 1.0
   * @default 2.0
   */
  Multiplier?: number;
}

export interface PitCounters {
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
