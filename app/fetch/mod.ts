import { Counter } from "../../core/mod.js";
import { Nanoseconds } from "../../core/nnduration/mod.js";

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
