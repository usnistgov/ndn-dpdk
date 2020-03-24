import * as iface from "../../iface/mod";

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
}

export interface FaceMgmt {
  List: {args: {}, reply: BasicInfo[]};
  Get: {args: IdArg, reply: FaceInfo};
  Create: {args: iface.Locator, reply: BasicInfo};
  Destroy: {args: iface.Locator, reply: {}};
}

export interface PortArg {
  Port: string;
}

export interface PortStatsArg extends PortArg {
  /**
   * @default false
   */
  Reset?: boolean;
}

export interface PortInfo {
  Name: string;
  NumaSocket: number;
  Active: boolean;
  ImplName?: string;
}

export interface EthFaceMgmt {
  ListPorts: {args: {}, reply: PortInfo[]};
  ListPortFaces: {args: PortArg, reply: BasicInfo[]};
  ReadPortStats: {args: PortStatsArg, reply: object};
}
