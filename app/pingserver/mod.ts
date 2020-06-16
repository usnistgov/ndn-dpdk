import * as pktqueue from "../../container/pktqueue/mod";
import { Milliseconds } from "../../core/nnduration/mod";
import * as ndni from "../../ndni/mod";

export interface Config {
  RxQueue?: pktqueue.ConfigPlain|pktqueue.ConfigDelay;
  Patterns: Pattern[];
  Nack: boolean;
}

export interface Pattern {
  Prefix: ndni.Name;
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
  Suffix: ndni.Name;

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
  Nack: ndni.NackReason;
}

interface ReplyTimeout {
  Timeout: true;
}

export type Reply = ReplyCommon & (ReplyData | ReplyNack | ReplyTimeout);
