import * as running_stat from "../../core/running_stat/mod.js";
import * as iface from "../../iface/mod.js";

export interface IdArg {
  /**
   * @TJS-type integer
   */
  Id: number;
}

export interface BasicInfo extends IdArg {
  Locator: iface.Locator;
}

export interface FaceInfo extends BasicInfo {
  IsDown: boolean;
  Counters: iface.Counters;
  ExCounters: any;
  Latency: running_stat.Snapshot;
}

export interface FaceMgmt {
  List: {args: {}, reply: BasicInfo[]};
  Get: {args: IdArg, reply: FaceInfo};
  Create: {args: iface.Locator, reply: BasicInfo};
  Destroy: {args: iface.Locator, reply: {}};
}
