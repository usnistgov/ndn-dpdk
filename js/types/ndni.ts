import type { NNMilliseconds, Uint } from "./core";

/** Name represented as canonical URI. */
export type Name = string;

/**
 * Interest template.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/ndni#InterestTemplateConfig>
 */
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
   * @default 255
   * @minimum 1
   * @maximum 255
   */
  hopLimit?: Uint;
}

/**
 * Data template.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/ndni#DataGenConfig>
 */
export interface DataGen {
  suffix?: Name;

  /**
   * @default 0
   */
  freshnessPeriod?: NNMilliseconds;

  /**
   * @default 0
   * @minimum 0
   */
  payloadLen?: Uint;
}
