import { FaceMgmt } from "./face";
import { FetchMgmt } from "./fetch";
import { FibMgmt } from "./fib";
import { PingClientMgmt } from "./pingclient";
import { VersionMgmt } from "./version";

export interface Mgmt {
  Face: FaceMgmt;
  Fetch: FetchMgmt;
  Fib: FibMgmt;
  PingClient: PingClientMgmt;
  Version: VersionMgmt;
}
