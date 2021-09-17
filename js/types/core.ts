/**
 * @asType integer
 * @minimum 0
 */
export type Uint = number;

/**
 * Non-negative counter.
 * It may be encoded as integer (uint32 or shorter) or string (uint64).
 */
export type Counter = Uint | string;

/**
 * @mininum 0.0
 * @maximum 1.0
 */
export type Ratio = number;

/**
 * Non-negative duration in milliseconds.
 * This can be either a non-negative integer in milliseconds or a string with any valid duration unit.
 */
export type NNMilliseconds = Uint | string;

/**
 * Non-negative duration in nanoseconds.
 * This can be either a non-negative integer in nanoseconds or a string with any valid duration unit.
 */
export type NNNanoseconds = Uint | string;

/** Snapshot from runningstat. */
export interface RunningStatSnapshot {
  /** Number of inputs. */
  count: Counter;

  /** Number of samples. */
  len: Counter;

  /** Minimum value. */
  min?: number;

  /** Maximum value. */
  max?: number;

  /** Mean. */
  mean?: number;

  /**
   * Variance of samples.
   * @minimum 0
   */
  variance?: number;

  /**
   * Standard deviation of samples.
   * @minimum 0
   */
  stdev?: number;

  /** Internal variable M1. */
  m1: number;

  /** Internal variable M2. */
  m2: number;
}
