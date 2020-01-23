import * as _ from "lodash";
import moment = require("moment");

import { Counter, Index } from "../../core/mod.js";
import { Nanoseconds } from "../../core/nnduration/mod.js";
import * as mgmt from "../../mgmt/mod.js";
import * as pingmgmt from "../../mgmt/pingmgmt/mod.js";

export interface TrafficGenCounters {
  raw: any; // raw counters
  nInterests: Counter;
  nData: Counter;
  satisfyRatio: number;
}

/**
 * Abstract traffic generator.
 */
export interface ITrafficGen {
  start(interval: Nanoseconds): Promise<void>;
  stop(rxDelay: moment.Duration): Promise<void>;
  readCounters(): Promise<TrafficGenCounters>;
}

/**
 * Use ndnping-dpdk as traffic generator, controlling over JSON-RPC API.
 */
export class NdnpingTrafficGen implements ITrafficGen {
  /**
   * Construct NdnpingTrafficGen that controls all clients in ndnping-dpdk program.
   */
  public static async create(rpc: mgmt.RpcClient = mgmt.makeMgmtClient()): Promise<ITrafficGen> {
    const self = new NdnpingTrafficGen(rpc);
    self.cList = await self.rpc.request<{}, Index[]>("PingClient.List", {});
    await self.stop(moment.duration(0));
    return self;
  }
  private rpc: mgmt.RpcClient;
  private cList: Index[];

  private constructor(rpc: mgmt.RpcClient) {
    this.rpc = rpc;
    this.cList = [];
  }

  public async start(interval: Nanoseconds): Promise<void> {
    await Promise.all(this.cList.map((index) =>
      this.rpc.request<pingmgmt.ClientStartArgs, {}>("PingClient.Start",
      { Index: index, Interval: interval, ClearCounters: true }),
    ));
  }

  public async stop(rxDelay: moment.Duration): Promise<void> {
    await Promise.all(this.cList.map((index) =>
      this.rpc.request<pingmgmt.ClientStopArgs, {}>("PingClient.Stop",
      { Index: index, RxDelay: rxDelay.asMilliseconds() * 1000000 }),
    ));
  }

  public async readCounters(): Promise<TrafficGenCounters> {
    const cnts = await Promise.all(this.cList.map((index) =>
      this.rpc.request<pingmgmt.IndexArg, pingmgmt.ClientCounters>("PingClient.ReadCounters",
      { Index: index }),
    ));
    const nInterests = _.sumBy(cnts, "NInterests");
    const nData = _.sumBy(cnts, "NData");
    return {
      raw: cnts,
      nInterests,
      nData,
      satisfyRatio: nData / nInterests,
    } as TrafficGenCounters;
  }
}
