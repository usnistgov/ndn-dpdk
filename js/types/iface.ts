import type { Counter, RunningStatSnapshot } from "./core";

/**
 * @TJS-type integer
 * @minimum 1
 * @maximum 65535
 */
export type FaceID = number;

export type FaceLocator = EthFaceLocator | SocketFaceLocator;

export interface EthFaceLocator {
  Scheme: "ether";
  Port: string;
  Local?: string;
  Remote?: string;

  /**
   * @items.type integer
   * @items.minimum 1
   * @items.maximum 4095
   */
  Vlan?: number[];
}

export interface SocketFaceLocator {
  Scheme: "udp"|"unixgram"|"tcp"|"unix";
  Local?: string;
  Remote: string;
}

export interface FaceCounters {
  RxFrames: Counter;
  RxOctets: Counter;

  DecodeErrs: Counter;
  ReassPackets: Counter;
  ReassDrops: Counter;

  RxInterests: Counter;
  RxData: Counter;
  RxNacks: Counter;

  InterestLatency: RunningStatSnapshot;
  DataLatency: RunningStatSnapshot;
  NackLatency: RunningStatSnapshot;

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

export interface CreateFaceConfig {
  EnableEth?: boolean;
  EthDisableRxFlow?: boolean;
  EthMtu?: number;
  EthRxqFrames?: number;
  EthTxqPkts?: number;
  EthTxqFrames?: number;

  EnableSock?: boolean;
  SockRxqFrames?: number;
  SockTxqPkts?: number;
  SockTxqFrames?: number;
}
