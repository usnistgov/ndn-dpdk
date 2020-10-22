import type { Index, NNNanoseconds } from "../core";
import type { TgConsumerCounters } from "../tg/mod";
import type { IndexArg } from "./common";

export interface PingClientMgmt {
  List: {args: {}; reply: Index[]};
  Start: {args: PingClientStartArgs; reply: {}};
  Stop: {args: PingClientStopArgs; reply: {}};
  ReadCounters: {args: IndexArg; reply: TgConsumerCounters};
}

export interface PingClientStartArgs extends IndexArg {
  Interval: NNNanoseconds;

  /**
   * @default false
   */
  ClearCounters?: boolean;
}

export interface PingClientStopArgs extends IndexArg {
  RxDelay?: NNNanoseconds;
}
