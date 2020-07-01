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
  Count: number;

  /**
   * Number of samples.
   * @TJS-type integer
   * @minimum 0
   */
  Len: number;

  /**
   * Minimum value.
   */
  Min?: number;

  /**
   * Maximum value.
   */
  Max?: number;

  /**
   * Mean.
   */
  Mean?: number;

  /**
   * Variance of samples.
   * @minimum 0
   */
  Variance?: number;

  /**
   * Standard deviation of samples.
   * @minimum 0
   */
  Stdev?: number;

  /**
   * Internal variable M1.
   */
  M1: number;

  /**
   * Internal variable M2.
   */
  M2: number;
}

/**
 * Name represented as canonical URI.
 */
export type Name = string;
