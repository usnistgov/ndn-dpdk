import type { Counter, NNNanoseconds, Uint } from "../core.js";
import type { InterestTemplate } from "../ndni.js";
import type { PktQueueConfig } from "../pktqueue.js";

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
  nTasks?: Uint;

  rxQueue?: PktQueueConfig;

  /**
   * @minimum 1
   * @default 65536
   */
  windowCapacity?: Uint;
}

export interface FetchTaskDef extends InterestTemplate {
  segmentBegin?: Uint;
  segmentEnd?: Uint;

  filename?: string;
  fileSize?: Uint;
  segmentLen?: Uint;
}

export interface FetchCounters {
  elapsed: NNNanoseconds;
  finished?: NNNanoseconds;
  lastRtt: NNNanoseconds;
  sRtt: NNNanoseconds;
  rto: NNNanoseconds;
  cwnd: Counter;
  nInFlight: Counter;
  nTxRetx: Counter;
  nRxData: Counter;
}
