import type { FaceMgmt } from "./face";
import type { FibMgmt } from "./fib";
import type { VersionMgmt } from "./version";

export interface Mgmt {
  Face: FaceMgmt;
  Fib: FibMgmt;
  Version: VersionMgmt;
}
