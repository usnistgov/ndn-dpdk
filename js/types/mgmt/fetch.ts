import type { Name, NNMilliseconds } from "../core";
import type { FetchCounters } from "../ping/mod";
import type { IndexArg } from "./common";

export interface FetchMgmt {
  List: {args: {}; reply: number[]};
  Benchmark: {args: FetchBenchmarkArgs; reply: FetchBenchmarkReply[]};
}

export interface FetchTemplate {
  Prefix: Name;
  CanBePrefix?: boolean;
  InterestLifetime?: NNMilliseconds;
}

export interface FetchBenchmarkArgs extends IndexArg {
  Templates: FetchTemplate[];
  Interval: NNMilliseconds;
  /**
   * @TJS-type integer
   * @minimum 2
   */
  Count: number;
}

export interface FetchBenchmarkReply {
  Counters: FetchCounters[];
  Goodput: number;
}
