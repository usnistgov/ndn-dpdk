import type { Counter, NNNanoseconds } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface FetchConfig {
  /**
     * @TJS-type integer
     * @minimum 1
     * @default 1
     */
  NThreads?: number;

  /**
     * @TJS-type integer
     * @minimum 1
     * @default 1
     */
  NProcs?: number;

  RxQueue?: PktQueueConfig;

  /**
     * @TJS-type integer
     * @minimum 1
     * @default 65536
     */
  WindowCapacity?: number;
}

export interface FetchCounters {
  Time: unknown;
  LastRtt: NNNanoseconds;
  SRtt: NNNanoseconds;
  Rto: NNNanoseconds;
  Cwnd: Counter;
  NInFlight: Counter;
  NTxRetx: Counter;
  NRxData: Counter;
}
