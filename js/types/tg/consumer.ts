import type { Counter, Name, NNMilliseconds, NNNanoseconds, RunningStatSnapshot } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface TgConsumerConfig {
  rxQueue?: PktQueueConfig;
  patterns: TgConsumerPattern[];
  interval: NNNanoseconds;
}

export interface TgConsumerPattern {
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

export interface TgConsumerCounters extends TgConsumerCounters.PacketCounters {
  nAllocError: Counter;
  rtt: RunningStatSnapshot;
  perPattern: TgConsumerCounters.PatternCounters[];
}

export namespace TgConsumerCounters {
  export interface PacketCounters {
    nInterests: Counter;
    nData: Counter;
    nNacks: Counter;
  }

  export interface PatternCounters extends PacketCounters {
    rtt: RunningStatSnapshot;
  }
}
