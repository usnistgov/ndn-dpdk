import type { Counter, RunningStatSnapshot } from "./core";
import type { LCore } from "./dpdk";
import type { FaceID } from "./iface";

export interface FwdpInputInfo {
  LCore: LCore;
  Faces: FaceID[];
}

export interface FwdpFwdInfo {
  LCore: LCore;

  InputInterest: FwdpFwdInfo.InputCounter;
  InputData: FwdpFwdInfo.InputCounter;
  InputNack: FwdpFwdInfo.InputCounter;
  InputLatency: RunningStatSnapshot;

  NNoFibMatch: Counter;
  NDupNonce: Counter;
  NSgNoFwd: Counter;
  NNackMismatch: Counter;

  HeaderMpUsage: Counter;
  IndirectMpUsage: Counter;
}

export namespace FwdpFwdInfo {
  export interface InputCounter {
    NDropped: Counter;
    NQueued: Counter;
    NCongMarks: Counter;
  }
}
