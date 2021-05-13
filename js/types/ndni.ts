import type { NNMilliseconds } from "./core";

/** Name represented as canonical URI. */
export type Name = string;

/** Interest template. */
export interface InterestTemplate {
  prefix: Name;

  /**
   * @default false
   */
  canBePrefix?: boolean;

  /**
   * @default false
   */
  mustBeFresh?: boolean;

  /**
   * @default 4000
   */
  interestLifetime?: NNMilliseconds;

  /**
   * @TJS-type integer
   * @default 255
   * @minimum 1
   * @maximum 255
   */
  hopLimit?: number;
}
