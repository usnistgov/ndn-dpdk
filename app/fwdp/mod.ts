import { Counter } from "../../core/mod.js";
import * as running_stat from "../../core/running_stat/mod.js";
import * as iface from "../../iface/mod.js";

export interface InputInfo {
  LCore: number;
  Faces: iface.FaceId[];

  NNameDisp: Counter;
  NTokenDisp: Counter;
  NBadToken: Counter;
}

export interface FwdInfo {
  LCore: number;
  NInterestDrops: Counter;
  NDataDrops: Counter;
  NNackDrops: Counter;
  NInterestCongMarks: Counter;
  NDataCongMarks: Counter;
  NNackCongMarks: Counter;
  InputLatency: running_stat.Snapshot;
  NNoFibMatch: Counter;
  NDupNonce: Counter;
  NSgNoFwd: Counter;
  NNackMismatch: Counter;
  HeaderMpUsage: Counter;
  IndirectMpUsage: Counter;
}
