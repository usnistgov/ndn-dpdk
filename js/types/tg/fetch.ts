import type { Counter, NNNanoseconds, Uint } from "../core";
import type { PktQueueConfig } from "../pktqueue";

export interface FetcherConfig {
  /**
   * @minimum 1
   * @default 1
   */
  nThreads?: Uint;

  /**
   * @minimum 1
   * @default 1
   */
  nProcs?: Uint;

  rxQueue?: PktQueueConfig;

  /**
   * @minimum 1
   * @default 65536
   */
  windowCapacity?: Uint;
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
