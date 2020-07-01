import type { FaceBasicInfo } from "./face";

export interface EthFaceMgmt {
  ListPorts: {args: {}; reply: EthPortInfo[]};
  ListPortFaces: {args: EthPortArg; reply: FaceBasicInfo[]};
  ReadPortStats: {args: EthPortStatsArg; reply: object};
}

export interface EthPortArg {
  Port: string;
}

export interface EthPortStatsArg extends EthPortArg {
  /**
   * @default false
   */
  Reset?: boolean;
}

export interface EthPortInfo {
  Name: string;
  NumaSocket: number;
  Active: boolean;
  ImplName?: string;
}
