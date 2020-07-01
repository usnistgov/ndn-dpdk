import { DpInfoMgmt } from "./dpinfo";
import { EthFaceMgmt } from "./ethface";
import { FaceMgmt } from "./face";
import { FetchMgmt } from "./fetch";
import { FibMgmt } from "./fib";
import { HrlogMgmt } from "./hrlog";
import { NdtMgmt } from "./ndt";
import { PingClientMgmt } from "./pingclient";
import { StrategyMgmt } from "./strategy";
import { VersionMgmt } from "./version";

export interface Mgmt {
  DpInfo: DpInfoMgmt;
  EthFace: EthFaceMgmt;
  Face: FaceMgmt;
  Fetch: FetchMgmt;
  Fib: FibMgmt;
  Hrlog: HrlogMgmt;
  Ndt: NdtMgmt;
  PingClient: PingClientMgmt;
  Strategy: StrategyMgmt;
  Version: VersionMgmt;
}
