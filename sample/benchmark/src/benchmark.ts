import type { ActivateFwArgs, ActivateGenArgs, EtherLocator, FaceLocator, FetchTaskDef, FileServerConfig, TgpConfig, VxlanLocator } from "@usnistgov/ndn-dpdk";
import assert from "minimalistic-assert";

import { GqlFwControl, GqlGenControl } from "./control";
import { hexPad, uniqueRandomVector } from "./util";

/**
 * Server-side environment variables.
 * See explanation in sample.env file.
 */
export interface ServerEnv {
  F_GQLSERVER: string;
  F_PORT_A: string;
  F_VLAN_A: number;
  F_HWADDR_A: string;
  F_PORT_B: string;
  F_VLAN_B: number;
  F_HWADDR_B: string;
  F_NUMA_PRIMARY: number;
  F_CORES_PRIMARY: readonly number[];
  F_CORES_SECONDARY: readonly number[];
  A_GQLSERVER: string;
  A_PORT_F: string;
  A_VLAN_F: number;
  A_HWADDR_F: string;
  A_NUMA_PRIMARY: number;
  A_CORES_PRIMARY: readonly number[];
  A_CORES_SECONDARY: readonly number[];
  A_FILESERVER_PATH: string;
  B_GQLSERVER: string;
  B_PORT_F: string;
  B_VLAN_F: number;
  B_HWADDR_F: string;
  B_NUMA_PRIMARY: number;
  B_CORES_PRIMARY: readonly number[];
  B_CORES_SECONDARY: readonly number[];
  B_FILESERVER_PATH: string;
}

/**
 * Benchmark options entered in BenchmarkOptionsEditor.
 * This type must be JSON serializable.
 */
export interface BenchmarkOptions {
  faceAScheme: BenchmarkOptions.FaceScheme;
  faceARxQueues: number;
  faceBScheme: BenchmarkOptions.FaceScheme;
  faceBRxQueues: number;
  nFwds: number;
  trafficDir: BenchmarkOptions.TrafficDir;
  producerKind: BenchmarkOptions.ProducerKind;
  nProducerThreads: number;
  nFlows: number;
  interestNameLen: number;
  dataMatch: BenchmarkOptions.DataMatch;
  payloadLen: number;
  segmentEnd: number;
  warmup: number;
  duration: number;
}
export namespace BenchmarkOptions {
  export type FaceScheme = "ether" | "vxlan" | "memif";
  export type TrafficDir = 2 | 1;
  export type ProducerKind = "pingserver" | "fileserver";
  export type DataMatch = "exact" | "prefix";
}

/**
 * Result of a single benchmark run.
 */
export interface BenchmarkResult {
  /** Run index. */
  i: number;

  /** End timestamp in milliseconds. */
  dt: number;

  /** Data retrieval duration. */
  duration: number;

  /** Throughput in pps. */
  pps: number;

  /** Goodput in bps. */
  bps: number;
}

/** Core benchmark logic. */
export class Benchmark {
  constructor(
      private readonly env: ServerEnv,
      private readonly opts: BenchmarkOptions,
      signal: AbortSignal,
  ) {
    this.cF = new GqlFwControl(env.F_GQLSERVER);
    this.cA = new GqlGenControl(env.A_GQLSERVER);
    this.cB = env.A_GQLSERVER === env.B_GQLSERVER ? this.cA : new GqlGenControl(env.B_GQLSERVER);
    signal.addEventListener("abort", () => {
      this.cF.close();
      this.cA.close();
      this.cB.close();
    });
  }

  private nRuns = 0;
  private readonly cF: GqlFwControl;
  private readonly cA: GqlGenControl;
  private readonly cB: GqlGenControl;
  private state = makeInitialState();

  /** Start forwarder and traffic generators for benchmark environment. */
  public async setup(): Promise<void> {
    await Promise.all([
      this.activateForwarder(),
      this.activateTrafficGen("A"),
      this.cA === this.cB ? undefined : this.activateTrafficGen("B"),
    ]);
    await Promise.all(tgNodeLabels.map((label) => this.startTrafficGen(label)));
  }

  private async activateForwarder(): Promise<void> {
    await this.cF.restart();

    const {
      faceARxQueues,
      faceBRxQueues,
      nFwds,
    } = this.opts;
    const alloc = this.env.F_CORES_PRIMARY.concat();
    const arg: ActivateFwArgs = {
      eal: {
        cores: [...this.env.F_CORES_PRIMARY, ...this.env.F_CORES_SECONDARY],
        lcoreMain: this.env.F_CORES_SECONDARY[0],
      },
      lcoreAlloc: {
        RX: alloc.splice(0, faceARxQueues + faceBRxQueues),
        TX: alloc.splice(0, 2),
        FWD: alloc.splice(0, nFwds),
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
    for (const label of tgNodeLabels) {
      const locator = await this.prepareLocator(this.cF, label, this.env[`F_PORT_${label}`], this.env[`F_VLAN_${label}`],
        this.env[`F_HWADDR_${label}`], this.env[`${label}_HWADDR_F`]);
      const face = await this.cF.createFace(locator);
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

  private async activateTrafficGen(label: TgNodeLabel): Promise<void> {
    const ctrl = this[`c${label}`];
    await ctrl.restart();

    const arg: ActivateGenArgs = {
      eal: {
        cores: [...this.env[`${label}_CORES_PRIMARY`], ...this.env[`${label}_CORES_SECONDARY`]],
        lcoreMain: this.env[`${label}_CORES_SECONDARY`][0],
      },
      mempool: {
        DIRECT: { capacity: 65535, dataroom: 9146 },
        INDIRECT: { capacity: 1048575 },
        PAYLOAD: { capacity: 16383 },
      },
    };
    await ctrl.activate("trafficgen", arg);
  }

  private async startTrafficGen(label: TgNodeLabel): Promise<void> {
    const ctrl = this[`c${label}`];
    const locator = await this.prepareLocator(ctrl, label, this.env[`${label}_PORT_F`], this.env[`${label}_VLAN_F`],
      this.env[`${label}_HWADDR_F`], this.env[`F_HWADDR_${label}`]);
    const result = await ctrl.startTrafficGen({
      face: locator,
      ...this.makeProducerConfig(label),
      fetcher: {
        nThreads: 1,
        nTasks: this.opts.nFlows,
      },
    });
    this.state.fetcher[label] = result.fetcher!;
    this.state.fileServerVersionBypassHi[label] = BigInt(result.fileServerVersionBypassHi ?? 0);
  }

  private async prepareLocator(ctrl: GqlFwControl | GqlGenControl, faceLabel: TgNodeLabel, pciAddr: string, vlan: number, local: string, remote: string): Promise<FaceLocator> {
    const isForwarder = ctrl === this.cF;
    const scheme = this.opts[`face${faceLabel}Scheme`];
    if (scheme === "memif") {
      return {
        scheme: "memif",
        role: isForwarder ? "server" : "client",
        socketName: "/run/ndn/ndndpdk-benchmark-memif.sock",
        id: faceLabel.codePointAt(0)!,
        dataroom: 9000,
      };
    }

    const port = await ctrl.createEthPort(pciAddr);
    const nRxQueues = this.opts[`face${faceLabel}RxQueues`];
    return {
      port,
      nRxQueues,
      local,
      remote,
      vlan,
      ...(scheme === "vxlan" ? vxlanLocatorFields : { scheme: "ether" }),
    };
  }

  private makeProducerConfig(label: string): { producer?: TgpConfig; fileServer?: FileServerConfig } {
    const {
      nFwds,
      trafficDir,
      producerKind,
      nProducerThreads,
      dataMatch,
      payloadLen,
    } = this.opts;

    if (!trafficDirProducers[trafficDir].includes(label as any)) {
      return {};
    }

    switch (producerKind) {
      case "pingserver": {
        const producer: TgpConfig = {
          nThreads: nProducerThreads,
          patterns: [],
        };
        for (let j = 0; j < nFwds; ++j) {
          producer.patterns.push({
            prefix: `/${label}/${j}`,
            replies: [{
              suffix: dataMatch === "exact" ? undefined : "/D",
              payloadLen,
              freshnessPeriod: 1,
            }],
          });
        }
        return { producer };
      }
      case "fileserver": {
        const fileServer: FileServerConfig = {
          nThreads: nProducerThreads,
          mounts: [{
            prefix: `/${label}`,
            path: this.env[`${label}_FILESERVER_PATH`],
          }],
          segmentLen: payloadLen,
          wantVersionBypass: true,
        };
        return { fileServer };
      }
    }
  }

  /** Run data retrievals once. */
  public async run(): Promise<BenchmarkResult> {
    const {
      payloadLen,
      warmup,
      duration,
    } = this.opts;

    await this.fetchStart();

    const abort = new AbortController();
    const t1 = warmup * 1e9;
    const t2 = (warmup + duration) * 1e9;
    const cnts = await Promise.all(this.listFetchTasks().map(([ctrl, id]) => ctrl.waitFetchProgress(id, abort.signal, t1, t2)));
    abort.abort();
    await Promise.all(this.eachTrafficDir((cLabel) => this[`c${cLabel}`].stopFetch(this.state.tasks[cLabel])));

    let totalPackets = 0;
    let totalSeconds = 0;
    for (const [cnt1, cnt2] of cnts) {
      totalPackets += Number(cnt2.nRxData) - Number(cnt1.nRxData);
      totalSeconds += (Number(cnt2.finished ?? cnt2.elapsed) - Number(cnt1.finished ?? cnt1.elapsed)) / 1e9;
    }
    const avgSeconds = totalSeconds / cnts.length;
    const pps = totalPackets / avgSeconds;
    return {
      i: this.nRuns++,
      dt: Date.now(),
      duration: avgSeconds,
      pps,
      bps: pps * payloadLen * 8,
    };
  }

  private eachTrafficDir<R>(f: (cLabel: TgNodeLabel, pLabel: TgNodeLabel) => R): R[] {
    return trafficDirProducers[this.opts.trafficDir].map((pLabel: TgNodeLabel) => {
      const cLabel = trafficDirProducerToConsumer[pLabel];
      return f(cLabel, pLabel);
    });
  }

  private async fetchStart(): Promise<void> {
    const fileVersionTime = BigInt.asUintN(24, BigInt(Math.trunc(Date.now() / 1000))) << 8n;
    const {
      nFwds,
      producerKind,
      nFlows,
      interestNameLen,
      dataMatch,
      segmentEnd,
    } = this.opts;
    const comp2 = uniqueRandomVector(nFlows, 1024);
    await Promise.all(this.eachTrafficDir(async (cLabel, pLabel) => {
      const tasks: FetchTaskDef[] = [];
      for (let j = 0; j < nFlows; ++j) {
        const prefix3 = `/${pLabel}/${j % nFwds}/${comp2[j]}`;
        switch (producerKind) {
          case "pingserver": {
            tasks.push({
              prefix: `${prefix3}${"/I".repeat(interestNameLen - 4)}`,
              canBePrefix: dataMatch === "prefix",
              mustBeFresh: true,
              segmentEnd,
            });
            break;
          }
          case "fileserver": {
            const fileVersion = (this.state.fileServerVersionBypassHi[pLabel] << 32n) | fileVersionTime | BigInt(j);
            assert(fileVersion > 0xFFFFFFFFn);
            const fileVersionCompV = hexPad(fileVersion, 16).replace(/[\dA-F]{2}/g, "%$&");
            tasks.push({
              prefix: `${prefix3}/54=${fileVersionCompV}`,
              segmentEnd,
            });
            break;
          }
        }
      }
      this.state.tasks[cLabel] = await this[`c${cLabel}`].startFetch(this.state.fetcher[cLabel], tasks);
    }));
  }

  private listFetchTasks(): Array<[ctrl: GqlGenControl, id: string]> {
    const list: Array<[ctrl: GqlGenControl, id: string]> = [];
    this.eachTrafficDir((cLabel) => {
      const ctrl = this[`c${cLabel}`];
      for (const id of this.state.tasks[cLabel]) {
        list.push([ctrl, id]);
      }
    });
    return list;
  }
}

interface State {
  /** TgNodeLabel => forwarder side face ID */
  face: Record<string, string>;
  /** whether NDT duplicates are detected */
  ndtDuplicate: boolean;
  /** TgNodeLabel => fetcher ID */
  fetcher: Record<string, string>;
  /** TgNodeLabel => fileserver versionBypassHi */
  fileServerVersionBypassHi: Record<string, bigint>;
  /** TgNodeLabel => fetcher task IDs */
  tasks: Record<string, string[]>;
}

function makeInitialState(): State {
  return {
    face: {},
    ndtDuplicate: false,
    fetcher: {},
    fileServerVersionBypassHi: {},
    tasks: {},
  };
}

type TgNodeLabel = "A" | "B";

const tgNodeLabels: readonly TgNodeLabel[] = ["A", "B"];

const trafficDirProducers: Record<BenchmarkOptions.TrafficDir, readonly TgNodeLabel[]> = {
  1: ["A"],
  2: ["A", "B"],
};

const trafficDirProducerToConsumer: Record<TgNodeLabel, TgNodeLabel> = {
  A: "B",
  B: "A",
};

const vxlanLocatorFields: Omit<VxlanLocator, Exclude<keyof EtherLocator, "scheme">> = {
  scheme: "vxlan",
  localIP: "192.168.118.0",
  remoteIP: "192.168.118.0",
  vxlan: 0,
  innerLocal: "02:00:00:ff:ff:ff",
  innerRemote: "02:00:00:ff:ff:ff",
};
