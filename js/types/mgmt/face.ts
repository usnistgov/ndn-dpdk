import type { FaceCounters, FaceLocator } from "../iface";
import type { IdArg } from "./common";

export interface FaceMgmt {
  List: {args: {}; reply: FaceBasicInfo[]};
  Get: {args: IdArg; reply: FaceInfo};
  Create: {args: FaceLocator; reply: FaceBasicInfo};
  Destroy: {args: FaceLocator; reply: {}};
}

export interface FaceBasicInfo extends IdArg {
  Locator: FaceLocator;
}

export interface FaceInfo extends FaceBasicInfo {
  IsDown: boolean;
  Counters: FaceCounters;
  ExCounters: any;
}
