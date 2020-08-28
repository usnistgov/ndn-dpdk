import { DpInfoMgmt } from "./dpinfo";
import { EthFaceMgmt } from "./ethface";
import { FaceMgmt } from "./face";
import { FetchMgmt } from "./fetch";
import { FibMgmt } from "./fib";
import { HrlogMgmt } from "./hrlog";
import { PingClientMgmt } from "./pingclient";
import { VersionMgmt } from "./version";

export interface Mgmt {
  DpInfo: DpInfoMgmt;
  EthFace: EthFaceMgmt;
  Face: FaceMgmt;
  Fetch: FetchMgmt;
  Fib: FibMgmt;
  Hrlog: HrlogMgmt;
  PingClient: PingClientMgmt;
  Version: VersionMgmt;
}
