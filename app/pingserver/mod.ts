import * as pktqueue from "../../container/pktqueue/mod.js";
import { Milliseconds } from "../../core/nnduration/mod.js";
import * as ndn from "../../ndn/mod.js";

export interface Config {
  RxQueue?: Omit<pktqueue.Config, "DisableCoDel">;
  Patterns: Pattern[];
  Nack: boolean;
}

export interface Pattern {
  Prefix: ndn.Name;
  Replies: Reply[];
}

interface ReplyCommon {
  /**
   * @TJS-type integer
   * @default 1
   * @minimum 1
   */
  Weight?: number;
}

interface ReplyData {
  Suffix: ndn.Name;

  /**
   * @default 0
   */
  FreshnessPeriod?: Milliseconds;

  /**
   * @TJS-type integer
   * @default 0
   * @minimum 0
   */
  PayloadLen?: number;
}

interface ReplyNack {
  Nack: ndn.NackReason;
}

interface ReplyTimeout {
  Timeout: true;
}

export type Reply = ReplyCommon & (ReplyData | ReplyNack | ReplyTimeout);
