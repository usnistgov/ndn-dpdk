import { Counter, NNDuration } from "../../core";
import * as ndn from "../../ndn";

export as namespace pingclient;

export interface Config {
  Patterns: Pattern[];
  Interval: NNDuration;
}

export interface Pattern {
  Weight: number;
  Prefix: ndn.Name;
  CanBePrefix: boolean;
  MustBeFresh: boolean;
  InterestLifetime: NNDuration;
  HopLimit: number;
  SeqNumOffset?: number;
}

interface PacketCounters {
  NInterests: Counter;
  NData: Counter;
  NNacks: Counter;
}

interface RttCounters {
  Min: NNDuration;
  Max: NNDuration;
  Avg: NNDuration;
  Stdev: NNDuration;
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
