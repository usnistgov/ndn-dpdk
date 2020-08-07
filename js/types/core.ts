/**
 * @TJS-type integer
 * @minimum 0
 */
export type Counter = number;

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
 * @TJS-type integer
 * @minimum 0
 */
export type NNMilliseconds = number;

/**
 * Non-negative duration in nanoseconds.
 * @TJS-type integer
 * @minimum 0
 */
export type NNNanoseconds = number;

/**
 * Snapshot from runningstat.
 */
export interface RunningStatSnapshot {
  /**
   * Number of inputs.
   * @TJS-type integer
   * @minimum 0
   */
  count: number;

  /**
   * Number of samples.
   * @TJS-type integer
   * @minimum 0
   */
  len: number;

  /**
   * Minimum value.
   */
  min?: number;

  /**
   * Maximum value.
   */
  max?: number;

  /**
   * Mean.
   */
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

  /**
   * Internal variable M1.
   */
  m1: number;

  /**
   * Internal variable M2.
   */
  m2: number;
}

/**
 * Name represented as canonical URI.
 */
export type Name = string;
