import { Counters as FetchCounters_ } from "../../app/fetch/mod.js";
import { Counters as ClientCounters_ } from "../../app/pingclient/mod.js";
import { Index } from "../../core/mod.js";
import { Milliseconds, Nanoseconds } from "../../core/nnduration/mod.js";

export interface IndexArg {
  Index: Index;
}

export interface ClientStartArgs extends IndexArg {
  Interval: Nanoseconds;

  /**
   * @default false
   */
  ClearCounters?: boolean;
}

export interface ClientStopArgs extends IndexArg {
  RxDelay?: Nanoseconds;
}

export type ClientCounters = ClientCounters_;

export interface PingClientMgmt {
  List: {args: {}, reply: Index[]};
  Start: {args: ClientStartArgs, reply: {}};
  Stop: {args: ClientStopArgs, reply: {}};
  ReadCounters: {args: IndexArg, reply: ClientCounters};
}

export type FetchCounters = FetchCounters_;

export interface FetchBenchmarkArgs {
  Index: Index;
  Warmup: Milliseconds;
  Interval: Milliseconds;
  Count: number;
}

export interface FetchBenchmarkReply {
  Counters: FetchCounters[];
}

export interface FetchMgmt {
  List: {args: {}, reply: Index[]};
  Benchmark: {args: FetchBenchmarkArgs, reply: FetchBenchmarkReply};
  ReadCounters: {args: IndexArg, reply: FetchCounters};
}
