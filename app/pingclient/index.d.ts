import { Counter } from "../../core";
import { Milliseconds, Nanoseconds } from "../../core/nnduration";
import * as ndn from "../../ndn";

export as namespace pingclient;

export interface Config {
  Patterns: Pattern[];
  Interval: Nanoseconds;
}

export interface Pattern {
  Weight: number;
  Prefix: ndn.Name;
  CanBePrefix: boolean;
  MustBeFresh: boolean;
  InterestLifetime: Milliseconds;
  HopLimit: number;
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
