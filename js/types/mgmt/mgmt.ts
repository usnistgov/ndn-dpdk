import { DpInfoMgmt } from "./dpinfo";
import { FaceMgmt } from "./face";
import { FetchMgmt } from "./fetch";
import { FibMgmt } from "./fib";
import { PingClientMgmt } from "./pingclient";
import { VersionMgmt } from "./version";

export interface Mgmt {
  DpInfo: DpInfoMgmt;
  Face: FaceMgmt;
  Fetch: FetchMgmt;
  Fib: FibMgmt;
  PingClient: PingClientMgmt;
  Version: VersionMgmt;
}
