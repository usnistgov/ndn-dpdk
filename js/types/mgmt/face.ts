import type { EtherLocator, FaceLocator } from "../iface.js";
import type { IdArg } from "./common.js";

export interface FaceMgmt {
  List: { args: {}; reply: FaceBasicInfo[] };
  Get: { args: IdArg; reply: FaceInfo };
  Create: { args: EtherLocator; reply: FaceBasicInfo };
  Destroy: { args: IdArg; reply: {} };
}

export interface FaceBasicInfo extends IdArg {
  Locator: FaceLocator;
}

export interface FaceInfo extends FaceBasicInfo {
  IsDown: boolean;
}
