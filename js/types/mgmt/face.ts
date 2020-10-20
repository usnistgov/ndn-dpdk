import type { EthFaceLocator, FaceLocator } from "../iface";
import type { IdArg } from "./common";

export interface FaceMgmt {
  List: {args: {}; reply: FaceBasicInfo[]};
  Get: {args: IdArg; reply: FaceInfo};
  Create: {args: EthFaceLocator; reply: FaceBasicInfo};
  Destroy: {args: IdArg; reply: {}};
}

export interface FaceBasicInfo extends IdArg {
  Locator: FaceLocator;
}

export interface FaceInfo extends FaceBasicInfo {
  IsDown: boolean;
}
