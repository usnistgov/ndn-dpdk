/**
 * Non-negative counter.
 * It may be encoded as integer (uint32 or shorter) or string (uint64).
 */
export type Counter = number | string;

/**
 * @TJS-type integer
 * @minimum 0
 */
export type Index = number;

/**
 * Base64 encoded binary blob.
 * @TJS-contentEncoding base64
 * @TJS-contentMediaType application/octet-stream
 */
export type Blob = string;

/**
 * Non-negative duration in milliseconds.
 * This can be either a non-negative integer in milliseconds or a string with any valid duration unit.
 */
export type NNMilliseconds = number | string;

/**
 * Non-negative duration in nanoseconds.
 * This can be either a non-negative integer in nanoseconds or a string with any valid duration unit.
 */
export type NNNanoseconds = number | string;

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

/** Name represented as canonical URI. */
export type Name = string;
