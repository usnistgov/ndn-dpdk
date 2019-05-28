import * as facemgmt from "./facemgmt";
import * as fibmgmt from "./fibmgmt";
import * as fwdpmgmt from "./fwdpmgmt";
import * as ndtmgmt from "./ndtmgmt";
import * as strategymgmt from "./strategymgmt";
import * as versionmgmt from "./versionmgmt";

export as namespace mgmt;

export interface Mgmt {
  Face: facemgmt.FaceMgmt;
  Fib: fibmgmt.FibMgmt;
  Fwdp: fwdpmgmt.FwdpMgmt;
  Ndt: ndtmgmt.NdtMgmt;
  Strategy: strategymgmt.StrategyMgmt;
  Version: versionmgmt.VersionMgmt;
}
