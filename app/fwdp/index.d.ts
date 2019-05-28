import { Counter } from "../../core";
import * as running_stat from "../../core/running_stat";
import * as iface from "../../iface";

export as namespace fwdp;

export interface InputInfo {
  LCore: number;
  Faces: iface.FaceId[];

  NNameDisp: Counter;
  NTokenDisp: Counter;
  NBadToken: Counter;
}

export interface FwdInfo {
  LCore: number;
  QueueCapacity: Counter;
  NQueueDrops: Counter;
  InputLatency: running_stat.Snapshot;
  NNoFibMatch: Counter;
  NDupNonce: Counter;
  NSgNoFwd: Counter;
  NNackMismatch: Counter;
  HeaderMpUsage: Counter;
  IndirectMpUsage: Counter;
}
