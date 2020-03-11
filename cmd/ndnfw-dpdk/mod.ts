import { InitConfig as BaseInitConfig } from "../../appinit/mod.js";
import { Config as PktQueueConfig } from "../../container/pktqueue/mod.js";
import { Config as NdtConfig } from "../../container/ndt/mod.js";
import { ConfigTemplate as FibConfig } from "../../container/fib/mod.js";

export interface FwdpInitConfig {
  FwdInterestQueue?: PktQueueConfig;
  FwdDataQueue?: PktQueueConfig;
  FwdNackQueue?: PktQueueConfig;
	LatencySampleFreq?: number;
	PcctCapacity?: number;
	CsCapMd?: number;
	CsCapMi?: number;
}

export interface InitConfig extends BaseInitConfig {
  Ndt?: NdtConfig;
  Fib?: FibConfig;
  Fwdp?: FwdpInitConfig;
}
