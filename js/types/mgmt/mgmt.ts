import { DpInfoMgmt } from "./dpinfo";
import { EthFaceMgmt } from "./ethface";
import { FaceMgmt } from "./face";
import { FetchMgmt } from "./fetch";
import { FibMgmt } from "./fib";
import { PingClientMgmt } from "./pingclient";
import { VersionMgmt } from "./version";

export interface Mgmt {
  DpInfo: DpInfoMgmt;
  EthFace: EthFaceMgmt;
  Face: FaceMgmt;
  Fetch: FetchMgmt;
  Fib: FibMgmt;
  PingClient: PingClientMgmt;
  Version: VersionMgmt;
}
