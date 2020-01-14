import { Milliseconds } from "../../core/nnduration";
import * as ndn from "../../ndn";

export as namespace pingserver;

export interface Config {
  Patterns: Pattern[];
  Nack: boolean;
}

export interface Pattern {
  Prefix: ndn.Name;
  Replies: Reply[];
}

interface ReplyCommon {
  Weight: number;
}

interface ReplyData {
  Suffix: ndn.Name;
  FreshnessPeriod: Milliseconds;
  PayloadLen: number;
}

interface ReplyNack {
  Nack: ndn.NackReason;
}

interface ReplyTimeout {
  Timeout: true;
}

export type Reply = ReplyCommon & (ReplyData | ReplyNack | ReplyTimeout);
