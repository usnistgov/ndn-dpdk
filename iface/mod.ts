import { Counter } from "../core/mod.js";
import * as runningStat from "../core/running_stat/mod.js";
import * as ethface from "./ethface/mod.js";
import * as mockface from "./mockface/mod.js";
import * as socketface from "./socketface/mod.js";

/**
 * @TJS-type integer
 * @minimum 1
 * @maximum 65535
 */
export type FaceId = number;

export type Locator = ethface.Locator | socketface.Locator | mockface.Locator;

export interface InOrderReassemblerCounters {
  Accepted: Counter;
  OutOfOrder: Counter;
  Delivered: Counter;
  Incomplete: Counter;
}

export interface Counters {
  RxFrames: Counter;
  RxOctets: Counter;

  L2DecodeErrs: Counter;
  Reass: InOrderReassemblerCounters;

  L3DecodeErrs: Counter;
  RxInterests: Counter;
  RxData: Counter;
  RxNacks: Counter;

  InterestLatency: runningStat.Snapshot;
  DataLatency: runningStat.Snapshot;
  NackLatency: runningStat.Snapshot;

  TxInterests: Counter;
  TxData: Counter;
  TxNacks: Counter;

  FragGood: Counter;
  FragBad: Counter;
  TxAllocErrs: Counter;
  TxDropped: Counter;
  TxFrames: Counter;
  TxOctets: Counter;
}
