import type { Counter, Name, NNMilliseconds, NNNanoseconds, RunningStatSnapshot } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface PingClientConfig {
  RxQueue?: PktQueueConfig;
  Patterns: PingClientPattern[];
  Interval: NNNanoseconds;
}

export interface PingClientPattern {
  /**
     * @TJS-type integer
     * @default 1
     * @minimum 1
     */
  Weight?: number;

  Prefix: Name;

  /**
     * @default false
     */
  CanBePrefix?: boolean;

  /**
     * @default false
     */
  MustBeFresh?: boolean;

  /**
     * @default 4000
     */
  InterestLifetime?: NNMilliseconds;

  /**
     * @TJS-type integer
     * @default 255
     * @minimum 1
     * @maximum 255
     */
  HopLimit?: number;

  /**
     * @TJS-type integer
     */
  SeqNumOffset?: number;
}

export interface PingClientCounters extends PingClientCounters.PacketCounters {
  NAllocError: Counter;
  Rtt: RunningStatSnapshot;
  PerPattern: PingClientCounters.PatternCounters[];
}

export namespace PingClientCounters {
  export interface PacketCounters {
    NInterests: Counter;
    NData: Counter;
    NNacks: Counter;
  }

  export interface PatternCounters extends PacketCounters {
    Rtt: RunningStatSnapshot;
  }
}
