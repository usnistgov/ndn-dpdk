import type { Counter, Name, NNMilliseconds, RunningStatSnapshot } from "../core";

/**
 * Traffic generator consumer pattern definition.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tgconsumer#Pattern>
 */
export interface TgcPattern {
  /**
     * @TJS-type integer
     * @default 1
     * @minimum 1
     */
  weight?: number;

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

  /**
   * @TJS-type integer
   */
  seqNumOffset?: number;
}

export interface TgcCounters extends TgcCounters.PacketCounters {
  nAllocError: Counter;
  rtt: RunningStatSnapshot;
  perPattern: TgcCounters.PatternCounters[];
}

export namespace TgcCounters {
  export interface PacketCounters {
    nInterests: Counter;
    nData: Counter;
    nNacks: Counter;
  }

  export interface PatternCounters extends PacketCounters {
    rtt: RunningStatSnapshot;
  }
}
