import { Counter } from "../core";
import * as running_stat from "../core/running_stat";
import * as ethface from "./ethface";
import * as mockface from "./mockface";
import * as socketface from "./socketface";

export as namespace iface;

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
  FragGood: Counter;
  FragBad: Counter;
  TxAllocErrs: Counter;
  TxQueued: Counter;
  TxDropped: Counter;
  TxInterests: Counter;
  TxData: Counter;
  TxNacks: Counter;
  TxFrames: Counter;
  TxOctets: Counter;
}
