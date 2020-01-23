import { Counter } from "../core/mod.js";
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
