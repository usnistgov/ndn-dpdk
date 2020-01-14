import { Index } from "../../core";
import { Nanoseconds } from "../../core/nnduration";
import { Counters as ClientCounters_ } from "../../app/pingclient";

export as namespace pingmgmt;

export interface IndexArg {
  Index: Index;
}

export interface ClientStartArgs {
  Index: Index;
  Interval: Nanoseconds;
  ClearCounters: boolean;
}

export interface ClientStopArgs {
  Index: Index;
  RxDelay: Nanoseconds;
}

export type ClientCounters = ClientCounters_;

export interface PingClientMgmt {
  List: {args: {}, reply: Index[]};
  Start: {args: ClientStartArgs, reply: {}};
  Stop: {args: ClientStopArgs, reply: {}};
  ReadCounters: {args: IndexArg, reply: ClientCounters};
}
