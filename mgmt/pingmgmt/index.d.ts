import { Index, NNDuration } from "../../core";
import * as ndnping from "../../app/ndnping";

export as namespace pingmgmt;

export interface IndexArg {
  Index: Index;
}

export interface ClientStartArgs {
  Index: Index;
  Interval: NNDuration;
  ClearCounters: boolean;
}

export interface ClientStopArgs {
  Index: Index;
  RxDelay: NNDuration;
}

export interface PingClientMgmt {
  List: {args: {}, reply: Index[]};
  Start: {args: ClientStartArgs, reply: {}};
  Stop: {args: ClientStopArgs, reply: {}};
  ReadCounters: {args: IndexArg, reply: ndnping.ClientCounters};
}
