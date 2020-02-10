import * as pktqueue from "../../container/pktqueue/mod.js";
import { Counter } from "../../core/mod.js";
import { Milliseconds, Nanoseconds } from "../../core/nnduration/mod.js";
import * as ndn from "../../ndn/mod.js";

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

interface RttCounters {
  Min: Nanoseconds;
  Max: Nanoseconds;
  Avg: Nanoseconds;
  Stdev: Nanoseconds;
}

interface PatternCounters extends PacketCounters {
  Rtt: RttCounters;
  NRttSamples: Counter;
}

export interface Counters extends PacketCounters {
  NAllocError: Counter;
  Rtt: RttCounters;
  PerPattern: PatternCounters[];
}
