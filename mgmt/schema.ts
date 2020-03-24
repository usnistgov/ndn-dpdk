import * as facemgmt from "./facemgmt/mod";
import * as fibmgmt from "./fibmgmt/mod";
import * as fwdpmgmt from "./fwdpmgmt/mod";
import * as hrlog from "./hrlog/mod";
import * as ndtmgmt from "./ndtmgmt/mod";
import * as pingmgmt from "./pingmgmt/mod";
import * as strategymgmt from "./strategymgmt/mod";
import * as versionmgmt from "./versionmgmt/mod";

export interface Mgmt {
  DpInfo: fwdpmgmt.DpInfoMgmt;
  EthFace: facemgmt.EthFaceMgmt;
  Face: facemgmt.FaceMgmt;
  Fetch: pingmgmt.FetchMgmt;
  Fib: fibmgmt.FibMgmt;
  Hrlog: hrlog.HrlogMgmt;
  Ndt: ndtmgmt.NdtMgmt;
  PingClient: pingmgmt.PingClientMgmt;
  Strategy: strategymgmt.StrategyMgmt;
  Version: versionmgmt.VersionMgmt;
}
