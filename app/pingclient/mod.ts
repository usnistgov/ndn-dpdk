import * as pktqueue from "../../container/pktqueue/mod";
import { Counter } from "../../core/mod";
import { Milliseconds, Nanoseconds } from "../../core/nnduration/mod";
import * as runningStat from "../../core/running_stat/mod";
import * as ndn from "../../ndn/mod";

export interface Config {
  RxQueue?: pktqueue.Config;
  Patterns: Pattern[];
  Interval: Nanoseconds;
}

export interface Pattern {
  /**
   * @TJS-type integer
   * @default 1
   * @minimum 1
   */
  Weight?: number;

  Prefix: ndn.Name;

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
  InterestLifetime?: Milliseconds;

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

interface PacketCounters {
  NInterests: Counter;
  NData: Counter;
  NNacks: Counter;
}

interface PatternCounters extends PacketCounters {
  Rtt: runningStat.Snapshot;
}

export interface Counters extends PacketCounters {
  NAllocError: Counter;
  Rtt: runningStat.Snapshot;
  PerPattern: PatternCounters[];
}
