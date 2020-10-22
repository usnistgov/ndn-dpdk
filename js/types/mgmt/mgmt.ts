import type { FaceMgmt } from "./face";
import type { FetchMgmt } from "./fetch";
import type { FibMgmt } from "./fib";
import type { PingClientMgmt } from "./pingclient";
import type { VersionMgmt } from "./version";

export interface Mgmt {
  Face: FaceMgmt;
  Fetch: FetchMgmt;
  Fib: FibMgmt;
  PingClient: PingClientMgmt;
  Version: VersionMgmt;
}
