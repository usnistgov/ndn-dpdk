import type { Counter, NNNanoseconds, RunningStatSnapshot, Uint } from "../core";
import type { DataGen, InterestTemplate } from "../ndni";
import type { PktQueueConfig } from "../pktqueue";

/**
 * Traffic generator consumer config.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tgconsumer#Config>
 */
export interface TgcConfig {
  rxQueue?: PktQueueConfig.Plain | PktQueueConfig.Delay;
  interval: NNNanoseconds;
  patterns: TgcPattern[];
}

/**
 * Traffic generator consumer pattern definition.
 * @see <https://pkg.go.dev/github.com/usnistgov/ndn-dpdk/app/tgconsumer#Pattern>
 */
export interface TgcPattern extends InterestTemplate {
  /**
   * @default 1
   * @minimum 1
   */
  weight?: Uint;

  /**
   * @default 0
   */
  seqNumOffset?: Uint;

  digest?: DataGen;
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
