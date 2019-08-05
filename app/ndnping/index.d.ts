import { Counter, NNDuration } from "../../core";
import * as iface from "../../iface";
import * as ndn from "../../ndn";

export as namespace ndnping;

export type AppConfig = TaskConfig[];

export interface TaskConfig {
  Face: iface.Locator;
  Client?: ClientConfig;
  Server?: ServerConfig;
}

export interface ClientConfig {
  Patterns: ClientPattern[];
  Interval: NNDuration;
}

export interface ClientPattern {
  Weight: number;
  Prefix: ndn.Name;
  CanBePrefix: boolean;
  MustBeFresh: boolean;
  InterestLifetime: NNDuration;
  HopLimit: number;
  SeqNumOffset?: number;
}

export interface ServerConfig {
  Patterns: ServerPattern[];
  Nack: boolean;
}

export interface ServerPattern {
  Prefix: ndn.Name;
  Replies: ServerReply[];
}

interface ServerReplyCommon {
  Weight: number;
}

interface ServerReplyData {
  Suffix: ndn.Name;
  FreshnessPeriod: NNDuration;
  PayloadLen: number;
}

interface ServerReplyNack {
  Nack: ndn.NackReason;
}

interface ServerReplyTimeout {
  Timeout: true;
}



export type ServerReply = ServerReplyCommon & (ServerReplyData | ServerReplyNack | ServerReplyTimeout);

interface ClientPacketCounters {
  NInterests: Counter;
  NData: Counter;
  NNacks: Counter;
}

interface ClientRttCounters {
  Min: NNDuration;
  Max: NNDuration;
  Avg: NNDuration;
  Stdev: NNDuration;
}

interface ClientPatternCounters extends ClientPacketCounters {
  Rtt: ClientRttCounters;
  NRttSamples: Counter;
}

export interface ClientCounters extends ClientPacketCounters {
  NAllocError: Counter;
  Rtt: ClientRttCounters;
  PerPattern: ClientPatternCounters[];
}
