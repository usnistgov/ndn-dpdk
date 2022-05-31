import type { FaceMgmt } from "./face.js";
import type { FibMgmt } from "./fib.js";
import type { VersionMgmt } from "./version.js";

/**
 * JSON-RPC 2.0 management API.
 * @deprecated New scripts should use GraphQL management.
 */
export interface Mgmt {
  Face: FaceMgmt;
  Fib: FibMgmt;
  Version: VersionMgmt;
}
