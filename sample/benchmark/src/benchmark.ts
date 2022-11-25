import { type ActivateFwArgs, type ActivateGenArgs, type EtherLocator, type FaceLocator, type FetchTaskDef, type FileServerConfig, type TgpConfig, type VxlanLocator } from "@usnistgov/ndn-dpdk";
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
  G_FILESERVER_PATH: string;
}

export interface BenchmarkOptions {
  faceAScheme: BenchmarkOptions.FaceScheme;
  faceARxQueues: number;
  faceBScheme: BenchmarkOptions.FaceScheme;
  faceBRxQueues: number;
  nFwds: number;
  producerKind: BenchmarkOptions.ProducerKind;
  interestNameLen: number;
  dataMatch: BenchmarkOptions.DataMatch;
  payloadLen: number;
  warmup: number;
  duration: number;
}
export namespace BenchmarkOptions {
  export type FaceScheme = "ether" | "vxlan" | "memif";
  export type ProducerKind = "pingserver" | "fileserver";
  export type DataMatch = "exact" | "prefix";
}

export interface BenchmarkState {
  face: Record<string, string>;
  ndtDuplicate: boolean;
  fetcher: Record<string, string>;
  fileServerVersionByPassBase: Record<string, bigint>;
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
    for (const [i, [label]] of DIRECTIONS.entries()) {
      const face = await this.cF.createFace(await this.prepareLocator(this.cF, this.env.F_PORTS[i]!, label));
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

    const arg: ActivateGenArgs = {
      eal: {
        cores: [...this.env.G_CORES_PRIMARY, ...this.env.G_CORES_SECONDARY],
        lcoreMain: this.env.G_CORES_SECONDARY[0],
      },
      mempool: {
        DIRECT: { capacity: 1048575, dataroom: 9146 },
        INDIRECT: { capacity: 2097151 },
        PAYLOAD: { capacity: 16383 },
      },
    };
    await this.cG.activate("trafficgen", arg);

    for (const [i, [label]] of DIRECTIONS.entries()) {
      while (!this.state.face[label]) {
        await delay(100);
      }
      const locator = await this.prepareLocator(this.cG, this.env.G_PORTS[i]!, label);
      const result = await this.cG.startTrafficGen({
        face: locator,
        ...this.makeProducerConfig(label),
        fetcher: {
          nThreads: 1,
          nTasks: this.opts.nFwds,
        },
      });
      this.state.fetcher[label] = result.fetcher!;
      this.state.fileServerVersionByPassBase[label] = BigInt(result.fileServerVersionByPassHi ?? 0) << 32n;
    }
  }

  private async prepareLocator(ctrl: GqlFwControl | GqlGenControl, portVlan: string, faceLabel: "A" | "B"): Promise<FaceLocator> {
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

    const [pciAddr, vlan] = portVlan.split("+");
    const port = await ctrl.createEthPort(pciAddr);
    const nRxQueues = this.opts[`face${faceLabel}RxQueues`];
    const macAddrLastOctet = faceLabel.codePointAt(0)!.toString(16).padStart(2, "0");
    return {
      port,
      nRxQueues,
      local: `02:00:00:00:${isForwarder ? "00" : "01"}:${macAddrLastOctet}`,
      remote: `02:00:00:00:${isForwarder ? "01" : "00"}:${macAddrLastOctet}`,
      vlan: vlan === undefined ? undefined : Number.parseInt(vlan, 10),
      ...(scheme === "vxlan" ? vxlanLocatorFields : { scheme: "ether" }),
    };
  }

  private makeProducerConfig(label: string): { producer?: TgpConfig; fileServer?: FileServerConfig } {
    const {
      nFwds,
      producerKind,
      dataMatch,
      payloadLen,
    } = this.opts;

    switch (producerKind) {
      case "pingserver": {
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
        return { producer };
      }
      case "fileserver": {
        const fileServer: FileServerConfig = {
          nThreads: 1,
          mounts: [{
            prefix: `/${label}`,
            path: this.env.G_FILESERVER_PATH,
          }],
          segmentLen: payloadLen,
          wantVersionBypass: true,
        };
        return { fileServer };
      }
    }
  }

  public async run(): Promise<Throughput> {
    const {
      payloadLen,
      warmup,
      duration,
    } = this.opts;

    await this.fetchStart();

    await delay(1000 * warmup);
    const cnts0 = await Promise.all(DIRECTIONS.map(([label]) => this.cG.getFetchProgress(this.state.tasks[label])));
    await delay(1000 * duration);
    const cnts1 = await Promise.all(DIRECTIONS.map(([label]) => this.cG.getFetchProgress(this.state.tasks[label])));
    await Promise.all(DIRECTIONS.map(([label]) => this.cG.stopFetch(this.state.tasks[label])));

    let totalPackets = 0;
    let totalElapsedSeconds = 0;
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

  private async fetchStart(): Promise<void> {
    const fileVersionTime = BigInt.asUintN(24, BigInt(Math.trunc(Date.now() / 1000))) << 8n;
    const {
      nFwds,
      producerKind,
      interestNameLen,
      dataMatch,
    } = this.opts;
    await Promise.all(DIRECTIONS.map(async ([label, dest]) => {
      const tasks: FetchTaskDef[] = [];
      for (let j = 0; j < nFwds; ++j) {
        switch (producerKind) {
          case "pingserver":
            tasks.push({
              prefix: `/${dest}/${j}${"/i".repeat(interestNameLen - 3)}`,
              canBePrefix: dataMatch === "prefix",
              mustBeFresh: true,
            });
            break;
          case "fileserver": {
            const fileVersion = this.state.fileServerVersionByPassBase[dest] | fileVersionTime | BigInt(j);
            const fileVersionCompV = fileVersion.toString(16).toUpperCase().padStart(16, "0").replace(/[\dA-F]{2}/g, "%$&");
            tasks.push({
              prefix: `/${dest}/${j}/32GB.bin/54=${fileVersionCompV}`,
            });
            break;
          }
        }
      }
      this.state.tasks[label] = await this.cG.startFetch(this.state.fetcher[label], tasks);
    }));
  }
}

const initialState: BenchmarkState = {
  face: {},
  ndtDuplicate: false,
  fetcher: {},
  fileServerVersionByPassBase: {},
  tasks: {},
};

const DIRECTIONS = [["A", "B"], ["B", "A"]] as const;

const vxlanLocatorFields: Omit<VxlanLocator, Exclude<keyof EtherLocator, "scheme">> = {
  scheme: "vxlan",
  localIP: "192.168.118.0",
  remoteIP: "192.168.118.0",
  vxlan: 0,
  innerLocal: "02:00:00:ff:ff:ff",
  innerRemote: "02:00:00:ff:ff:ff",
};
