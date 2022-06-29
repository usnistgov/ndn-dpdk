import { type ActivateFwArgs, type ActivateGenArgs, type EtherLocator, type FaceLocator, type FetcherConfig, type FetchTaskDef, type TgpConfig, type VxlanLocator } from "@usnistgov/ndn-dpdk";
import delay from "delay";

import { GqlFwControl, GqlGenControl } from "./control";

export interface ServerEnv {
  F_GQLSERVER: string;
  F_PORTS: string[];
  F_NUMA_PRIMARY: number;
  F_CORES_PRIMARY: number[];
  F_CORES_SECONDARY: number[];
  G_GQLSERVER: string;
  G_PORTS: string[];
  G_NUMA_PRIMARY: number;
  G_CORES_PRIMARY: number[];
  G_CORES_SECONDARY: number[];
}

export interface BenchmarkOptions {
  faceScheme: "ether" | "vxlan";
  faceRxQueues: number;
  nFwds: number;
  interestNameLen: number;
  dataMatch: "exact" | "prefix";
  payloadLen: number;
  warmup: number;
  duration: number;
}

export interface BenchmarkState {
  face: Record<string, string>;
  ndtDuplicate: boolean;
  fetcher: Record<string, string>;
  tasks: Record<string, string[]>;
}

export interface Throughput {
  pps: number;
  bps: number;
}

export class Benchmark {
  constructor(
      private readonly env: ServerEnv,
      private readonly opts: BenchmarkOptions,
      signal: AbortSignal,
  ) {
    this.cF = new GqlFwControl(env.F_GQLSERVER);
    this.cG = new GqlGenControl(env.G_GQLSERVER);
    this.state = JSON.parse(JSON.stringify(initialState));
    signal.addEventListener("abort", () => {
      this.cF.close();
      this.cG.close();
    });
  }

  private readonly cF: GqlFwControl;
  private readonly cG: GqlGenControl;
  private state: BenchmarkState;

  public async setupForwarder(): Promise<void> {
    await this.cF.restart();

    const {
      faceScheme,
      faceRxQueues,
      nFwds,
    } = this.opts;
    const arg: ActivateFwArgs = {
      eal: {
        cores: [...this.env.F_CORES_PRIMARY, ...this.env.F_CORES_SECONDARY],
        lcoreMain: this.env.F_CORES_SECONDARY[0],
        memPerNuma: { [this.env.F_NUMA_PRIMARY]: 16384 },
      },
      lcoreAlloc: {
        RX: this.env.F_CORES_PRIMARY.slice(-2 * faceRxQueues),
        TX: this.env.F_CORES_PRIMARY.slice(0, 2),
        FWD: this.env.F_CORES_PRIMARY.slice(2, 2 + nFwds),
        CRYPTO: [this.env.F_CORES_SECONDARY[1]],
      },
      mempool: {
        DIRECT: { capacity: 1048575, dataroom: 9146 },
        INDIRECT: { capacity: 2097151 },
      },
      ndt: { prefixLen: 2 },
      fib: { startDepth: 4 },
      pcct: {
        pcctCapacity: 65535,
        csMemoryCapacity: 4096,
        csIndirectCapacity: 4096,
      },
      fwdInterestQueue: { dequeueBurstSize: 32 },
      fwdDataQueue: { dequeueBurstSize: 64 },
      fwdNackQueue: { dequeueBurstSize: 64 },
    };
    await this.cF.activate("forwarder", arg);

    const seenNdtIndices = new Set<number>();
    for (const [i, [label]] of DIRECTIONS.entries()) {
      const port = await this.cF.createEthPort(this.env.F_PORTS[i]!);
      const face = await this.cF.createFace({
        port,
        nRxQueues: faceRxQueues,
        local: macAddr("F", label),
        remote: macAddr("G", label),
        ...(faceScheme === "vxlan" ? vxlanLocatorFields : { scheme: "ether" }),
      });
      this.state.face[label] = face;

      for (let j = 0; j < nFwds; ++j) {
        const name = `/${label}/${j}`;
        await this.cF.insertFibEntry(name, face);

        const index = await this.cF.updateNdt(name, j % nFwds);
        this.state.ndtDuplicate ||= seenNdtIndices.has(index);
        seenNdtIndices.add(index);
      }
    }
  }

  public async setupTrafficGen(): Promise<void> {
    await this.cG.restart();

    const {
      faceScheme,
      faceRxQueues,
      nFwds,
      dataMatch,
      payloadLen,
    } = this.opts;
    const arg: ActivateGenArgs = {
      eal: {
        cores: [...this.env.G_CORES_PRIMARY, ...this.env.G_CORES_SECONDARY],
        lcoreMain: this.env.G_CORES_SECONDARY[0],
        memPerNuma: { [this.env.G_NUMA_PRIMARY]: 16384 },
      },
      mempool: {
        DIRECT: { capacity: 1048575, dataroom: 9146 },
        INDIRECT: { capacity: 2097151 },
      },
    };
    await this.cG.activate("trafficgen", arg);

    for (const [i, [label]] of DIRECTIONS.entries()) {
      const port = await this.cG.createEthPort(this.env.G_PORTS[i]!);
      const locator: FaceLocator = {
        port,
        nRxQueues: faceRxQueues,
        local: macAddr("G", label),
        remote: macAddr("F", label),
        ...(faceScheme === "vxlan" ? vxlanLocatorFields : { scheme: "ether" }),
      };
      const producer: TgpConfig = {
        nThreads: 1,
        patterns: [],
      };
      for (let j = 0; j < nFwds; ++j) {
        producer.patterns.push({
          prefix: `/${label}/${j}`,
          replies: [{
            suffix: dataMatch === "exact" ? undefined : "/d",
            payloadLen,
            freshnessPeriod: 1,
          }],
        });
      }
      const fetcher: FetcherConfig = {
        nThreads: 1,
        nTasks: nFwds,
      };
      const result = await this.cG.startTrafficGen(locator, producer, fetcher);
      this.state.fetcher[label] = result.fetcher;
    }
  }

  public async run(): Promise<Throughput> {
    const {
      nFwds,
      interestNameLen,
      dataMatch,
      payloadLen,
      warmup,
      duration,
    } = this.opts;
    await Promise.all(DIRECTIONS.map(async ([label, dest]) => {
      const tasks: FetchTaskDef[] = [];
      for (let j = 0; j < nFwds; ++j) {
        tasks.push({
          prefix: `/${dest}/${j}${"/i".repeat(interestNameLen - 3)}`,
          canBePrefix: dataMatch === "prefix",
          mustBeFresh: true,
        });
      }
      this.state.tasks[label] = await this.cG.startFetch(this.state.fetcher[label], tasks);
    }));
    await delay(1000 * warmup);
    const cnts0 = await Promise.all(DIRECTIONS.map(([label]) => this.cG.getFetchProgress(this.state.tasks[label])));
    await delay(1000 * duration);
    const cnts1 = await Promise.all(DIRECTIONS.map(([label]) => this.cG.getFetchProgress(this.state.tasks[label])));
    await Promise.all(DIRECTIONS.map(([label]) => this.cG.stopFetch(this.state.tasks[label])));

    let totalPackets = 0; let
      totalElapsedSeconds = 0;
    for (const [i, cnt0d] of cnts0.entries()) {
      for (const [j, cnt0] of cnt0d.entries()) {
        const cnt1 = cnts1[i]![j]!;
        totalPackets += Number(cnt1.nRxData) - Number(cnt0.nRxData);
        totalElapsedSeconds += (Number(cnt1.elapsed) - Number(cnt0.elapsed)) / 1e9;
      }
    }
    const avgElapsedSeconds = totalElapsedSeconds / cnts0.length / cnts0[0]!.length;
    const pps = totalPackets / avgElapsedSeconds;
    return {
      pps,
      bps: pps * payloadLen * 8,
    };
  }
}

const initialState: BenchmarkState = {
  face: {},
  ndtDuplicate: false,
  fetcher: {},
  tasks: {},
};

const DIRECTIONS = [["A", "B"], ["B", "A"]];

function hexDigit(s: string): string {
  return s.codePointAt(0)!.toString(16).padStart(2, "0");
}

function macAddr(node: string, intf: string): string {
  return `02:00:00:00:${hexDigit(node)}:${hexDigit(intf)}`;
}

const vxlanLocatorFields: Omit<VxlanLocator, Exclude<keyof EtherLocator, "scheme">> = {
  scheme: "vxlan",
  localIP: "192.168.118.0",
  remoteIP: "192.168.118.0",
  vxlan: 2,
  innerLocal: "02:00:00:ff:ff:ff",
  innerRemote: "02:00:00:ff:ff:ff",
};
