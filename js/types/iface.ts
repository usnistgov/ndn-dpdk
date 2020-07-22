import type { Counter, NNMilliseconds, RunningStatSnapshot } from "./core";

/**
 * @TJS-type integer
 * @minimum 1
 * @maximum 65535
 */
export type FaceID = number;

export type FaceLocator = EthFaceLocator | SocketFaceLocator;

export interface FaceConfig {
  /**
   * @TJS-type integer
   * @minimum 4
   * @maximum 8192
   * @default 64
   */
  reassemblerCapacity?: number;

  /**
   * @TJS-type integer
   * @minimum 256
   * @default 1024
   */
  outputQueueSize?: number;

  /**
   * @TJS-type integer
   * @minimum 1280
   * @maximum 65000
   */
  mtu?: number;
}

export interface EthFaceLocator {
  scheme: "ether";
  local: string;
  remote: string;

  /**
   * @TJS-type integer
   * @minimum 1
   * @maximum 4095
   */
  vlan?: number;

  port?: string;
  portConfig?: EthPortConfig;
  rxQueueIDs?: number[];
}

export interface EthPortConfig extends FaceConfig {
  disableRxFlow?: boolean;

  /**
   * @TJS-type integer
   * @minimum 256
   * @default 8192
   */
  rxQueueSize?: number;

  /**
   * @TJS-type integer
   * @minimum 256
   * @default 8192
   */
  txQueueSize?: number;

  noSetMTU?: boolean;
}

export interface SocketFaceLocator {
  scheme: "udp"|"tcp"|"unix";
  local?: string;
  remote: string;
}

export interface SocketFaceConfig extends FaceConfig {
  /**
   * @TJS-type integer
   * @minimum 256
   * @default 4096
   */
  rxGroupQueueSize?: number;

  rxQueueSize?: number;
  txQueueSize?: number;
  redialBackoffInitial?: NNMilliseconds;
  redialBackoffMaximum?: NNMilliseconds;
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
  EnableSock?: boolean;
}
