import * as pktqueue from "../../container/pktqueue/mod.js";
import { Counter } from "../../core/mod.js";
import { Nanoseconds } from "../../core/nnduration/mod.js";

export interface Config {
  RxQueue?: pktqueue.Config;

  /**
   * @TJS-type integer
   * @minimum 1
   * @default 65536
   */
  WindowCapacity?: number;
}

export interface Counters {
  Time: unknown;
  LastRtt: Nanoseconds;
  SRtt: Nanoseconds;
  Rto: Nanoseconds;
  Cwnd: Counter;
  NInFlight: Counter;
  NTxRetx: Counter;
  NRxData: Counter;
}
