import * as _ from "lodash";
import moment = require("moment");

import * as ndnping from "../../app/ndnping";
import { Counter, Index, NNDuration } from "../../core";
import { RpcClient } from "../../mgmt";
import * as pingmgmt from "../../mgmt/pingmgmt";

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
  start(interval: NNDuration): Promise<void>;
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
  public static async create(rpc: RpcClient): Promise<ITrafficGen> {
    const self = new NdnpingTrafficGen(rpc);
    self.cList = await self.rpc.request<{}, Index[]>("PingClient.List", {});
    await self.stop(moment.duration(0));
    return self;
  }
  private rpc: RpcClient;
  private cList: Index[];

  private constructor(rpc: RpcClient) {
    this.rpc = rpc;
    this.cList = [];
  }

  public async start(interval: NNDuration): Promise<void> {
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
      this.rpc.request<pingmgmt.IndexArg, ndnping.ClientCounters>("PingClient.ReadCounters",
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
