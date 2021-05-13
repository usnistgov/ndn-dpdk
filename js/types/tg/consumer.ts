import type { Counter, RunningStatSnapshot } from "../core";
import type { InterestTemplate } from "../ndni";

/**
 * Traffic generator consumer pattern definition.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tgconsumer#Pattern>
 */
export interface TgcPattern extends InterestTemplate {
  /**
     * @TJS-type integer
     * @default 1
     * @minimum 1
     */
  weight?: number;

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
