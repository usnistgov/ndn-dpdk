import type { FaceMgmt } from "./face";
import type { FibMgmt } from "./fib";
import type { VersionMgmt } from "./version";

/**
 * JSON-RPC 2.0 management API.
 * @deprecated New scripts should use GraphQL management.
 */
export interface Mgmt {
  Face: FaceMgmt;
  Fib: FibMgmt;
  Version: VersionMgmt;
}
