import type { Counter, NNMilliseconds } from "./core";

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
  scheme: "ether"|"memif";
  local: string;
  remote: string;

  /**
   * @TJS-type integer
   * @minimum 1
   * @maximum 4095
   */
  vlan?: number;

  memif?: MemifLocator;
  port?: string;
  portConfig?: EthPortConfig;
  rxQueueIDs?: number[];
}

export interface MemifLocator {
  socketName: string;

  /**
   * @TJS-type integer
   * @minimum 0
   * @maximum 4294967295
   */
  id: number;

  /**
   * @TJS-type integer
   * @minimum 512
   * @maximum 65535
   * @default 2048
   */
  dataroom?: number;

  /**
   * @TJS-type integer
   * @minimum 2
   * @maximum 16384
   * @default 1024
   */
  ringCapacity?: number;
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
  rxFrames: Counter;
  rxOctets: Counter;

  decodeErrs: Counter;
  reassPackets: Counter;
  reassDrops: Counter;

  rxInterests: Counter;
  rxData: Counter;
  rxNacks: Counter;

  txInterests: Counter;
  txData: Counter;
  txNacks: Counter;

  fragGood: Counter;
  fragBad: Counter;
  txAllocErrs: Counter;
  txDropped: Counter;
  txFrames: Counter;
  txOctets: Counter;
}
