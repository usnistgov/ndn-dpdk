export * from "./rpc-client";

import * as facemgmt from "./facemgmt/mod.js";
import * as fibmgmt from "./fibmgmt/mod.js";
import * as fwdpmgmt from "./fwdpmgmt/mod.js";
import * as hrlog from "./hrlog/mod.js";
import * as ndtmgmt from "./ndtmgmt/mod.js";
import * as pingmgmt from "./pingmgmt/mod.js";
import * as strategymgmt from "./strategymgmt/mod.js";
import * as versionmgmt from "./versionmgmt/mod.js";

export interface Mgmt {
  EthFace: facemgmt.EthFaceMgmt;
  Face: facemgmt.FaceMgmt;
  Fetch: pingmgmt.FetchMgmt;
  Fib: fibmgmt.FibMgmt;
  Fwdp: fwdpmgmt.FwdpMgmt;
  Hrlog: hrlog.HrlogMgmt;
  Ndt: ndtmgmt.NdtMgmt;
  PingClient: pingmgmt.PingClientMgmt;
  Strategy: strategymgmt.StrategyMgmt;
  Version: versionmgmt.VersionMgmt;
}
