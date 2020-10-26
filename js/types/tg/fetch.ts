import type { Counter, NNNanoseconds } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface FetchConfig {
  /**
   * @TJS-type integer
   * @minimum 1
   * @default 1
   */
  nThreads?: number;

  /**
   * @TJS-type integer
   * @minimum 1
   * @default 1
   */
  nProcs?: number;

  rxQueue?: PktQueueConfig;

  /**
   * @TJS-type integer
   * @minimum 1
   * @default 65536
   */
  windowCapacity?: number;
}

export interface FetchCounters {
  time: unknown;
  lastRtt: NNNanoseconds;
  sRtt: NNNanoseconds;
  rto: NNNanoseconds;
  cwnd: Counter;
  nInFlight: Counter;
  nTxRetx: Counter;
  nRxData: Counter;
}
