import type { NNNanoseconds } from "./core";

/**
 * PIT suppression configuration.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/container/pit#SuppressConfig>
 */
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
