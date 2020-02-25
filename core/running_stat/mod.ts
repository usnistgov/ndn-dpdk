export interface Snapshot {
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
}
