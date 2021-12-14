import type { ActivateFwArgs, ActivateGenArgs, EtherLocator, FaceLocator, FetchCounters, InterestTemplate, TgConfig, VxlanLocator } from "@usnistgov/ndn-dpdk";
import delay from "delay";
import { gql, GraphQLClient } from "graphql-request";

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
  duration: number;
}

export interface BenchmarkState {
  face: Record<string, string>;
  ndtDuplicate: boolean;
  fetcher: Record<string, string>;
}

export type BenchmarkResult = Record<string, FetchCounters[][]>;

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
    this.cF = new GraphQLClient(env.F_GQLSERVER, { signal });
    this.cG = new GraphQLClient(env.G_GQLSERVER, { signal });
    this.state = JSON.parse(JSON.stringify(initialState));
  }

  private readonly cF: GraphQLClient;
  private readonly cG: GraphQLClient;
  private state: BenchmarkState;

  public async setupForwarder(): Promise<void> {
    await restart(this.cF);

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
        csDirectCapacity: 4096,
        csIndirectCapacity: 4096,
      },
      fwdInterestQueue: { dequeueBurstSize: 32 },
      fwdDataQueue: { dequeueBurstSize: 64 },
      fwdNackQueue: { dequeueBurstSize: 64 },
    };
    await this.cF.request(gql`
      mutation activate($arg: JSON!) {
        activate(forwarder: $arg)
      }
    `, { arg });

    const seenNdtIndices = new Set<number>();
    for (const [i, [label]] of DIRECTIONS.entries()) {
      const port = await createEthPort(this.cF, this.env.F_PORTS[i]!);
      const face = await createFace(this.cF, {
        port,
        maxRxQueues: faceRxQueues,
        local: macAddr("F", label),
        remote: macAddr("G", label),
        ...(faceScheme === "vxlan" ? vxlanLocatorFields : { scheme: "ether" }),
      });
      this.state.face[label] = face;

      for (let j = 0; j < nFwds; ++j) {
        const name = `/${label}/${j}`;
        await insertFibEntry(this.cF, name, face);

        const index = await updateNdt(this.cF, name, j % nFwds);
        this.state.ndtDuplicate ||= seenNdtIndices.has(index);
        seenNdtIndices.add(index);
      }
    }
  }

  public async setupTrafficGen(): Promise<void> {
    await restart(this.cG);

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
    await this.cG.request(gql`
      mutation activate($arg: JSON!) {
        activate(trafficgen: $arg)
      }
    `, { arg });

    for (const [i, [label]] of DIRECTIONS.entries()) {
      const port = await createEthPort(this.cG, this.env.G_PORTS[i]!);

      const cfg: TgConfig = {
        face: {
          port,
          maxRxQueues: faceRxQueues,
          local: macAddr("G", label),
          remote: macAddr("F", label),
          ...(faceScheme === "vxlan" ? vxlanLocatorFields : { scheme: "ether" }),
        },
        producer: {
          nThreads: 1,
          patterns: [],
        },
        fetcher: {
          nThreads: 1,
          nProcs: nFwds,
        },
      };
      for (let j = 0; j < nFwds; ++j) {
        cfg.producer!.patterns.push({
          prefix: `/${label}/${j}`,
          replies: [{
            suffix: dataMatch === "exact" ? undefined : "/d",
            payloadLen,
            freshnessPeriod: 1,
          }],
        });
      }

      const { startTrafficGen: { fetcher: { id } } } = await this.cG.request<{
        startTrafficGen: { fetcher: { id: string } };
      }>(gql`
        mutation startTrafficGen(
          $face: JSON!
          $producer: TgProducerConfigInput
          $fetcher: FetcherConfigInput
        ) {
          startTrafficGen(
            face: $face
            producer: $producer
            fetcher: $fetcher
          ) {
            fetcher { id }
          }
        }
      `, cfg);
      this.state.fetcher[label] = id;
    }
  }

  public async run(): Promise<BenchmarkResult> {
    const {
      nFwds,
      interestNameLen,
      dataMatch,
      duration,
    } = this.opts;
    const result = Object.fromEntries(await Promise.all(DIRECTIONS.map(async ([label, dest]) => {
      const templates: InterestTemplate[] = [];
      for (let j = 0; j < nFwds; ++j) {
        templates.push({
          prefix: `/${dest}/${j}${"/i".repeat(interestNameLen - 3)}`,
          canBePrefix: dataMatch === "prefix",
          mustBeFresh: true,
        });
      }
      const { runFetchBenchmark } = await this.cG.request<{ runFetchBenchmark: any }>(gql`
        mutation runFetchBenchmark($fetcher: ID!, $templates: [InterestTemplateInput!]!, $interval: NNNanoseconds!, $count: Int!) {
          runFetchBenchmark(fetcher: $fetcher, templates: $templates, interval: $interval, count: $count)
        }
      `, {
        fetcher: this.state.fetcher[label],
        templates,
        interval: "1s",
        count: 1 + duration,
      });
      return [label, runFetchBenchmark];
    })));
    await delay(1000);
    return result;
  }

  public computeThroughput(a: BenchmarkResult | readonly FetchCounters[] | readonly FetchCounters[][]): Throughput {
    if (Array.isArray(a)) {
      if (Array.isArray(a[0])) {
        // eslint-disable-next-line unicorn/no-array-method-this-argument
        return sumThroughput((a as FetchCounters[][]).map(this.computeThroughput, this));
      }

      const { payloadLen, duration } = this.opts;
      const r = a as FetchCounters[];
      const pps = (Number(r.at(-1)!.nRxData) - Number(r[0].nRxData)) / duration;
      return { pps, bps: pps * payloadLen * 8 };
    }

    // eslint-disable-next-line unicorn/no-array-method-this-argument
    return sumThroughput(Object.values(a as BenchmarkResult).map(this.computeThroughput, this));
  }
}

const initialState: BenchmarkState = {
  face: {},
  ndtDuplicate: false,
  fetcher: {},
};

const DIRECTIONS = [["A", "B"], ["B", "A"]];

async function restart(c: GraphQLClient) {
  await c.request(gql`mutation { shutdown(restart: true) }`);
  await delay(5000);
  for (let i = 0; i < 30; ++i) {
    try {
      await delay(1000);
      await c.request(gql`{ version { version } }`);
      return;
    } catch {}
  }
  throw new Error("restart timeout");
}

function hexDigit(s: string): string {
  return s.charCodeAt(0).toString(16).padStart(2, "0").toUpperCase();
}

function macAddr(node: string, intf: string): string {
  return `02:00:00:00:${hexDigit(node)}:${hexDigit(intf)}`;
}

const vxlanLocatorFields: Omit<VxlanLocator, keyof EtherLocator> & Pick<VxlanLocator, "scheme"> = {
  scheme: "vxlan",
  localIP: "192.168.118.0",
  remoteIP: "192.168.118.0",
  vxlan: 2,
  innerLocal: "02:00:00:ff:ff:ff",
  innerRemote: "02:00:00:ff:ff:ff",
};

async function createEthPort(client: GraphQLClient, pciAddr: string): Promise<string> {
  const { createEthPort: { id } } = await client.request<{
    createEthPort: { id: string };
  }>(gql`
    mutation createEthPort(
      $pciAddr: String
    ) {
      createEthPort(
        driver: PCI
        pciAddr: $pciAddr
        mtu: 9000
        rxFlowQueues: 2
      ) {
        id
      }
    }
  `, { pciAddr });
  return id;
}

async function createFace(client: GraphQLClient, locator: FaceLocator): Promise<string> {
  const { createFace: { id } } = await client.request<{
    createFace: { id: string };
  }>(gql`
    mutation createFace($locator: JSON!) {
      createFace(locator: $locator) {
        id
      }
    }
  `, { locator });
  return id;
}

async function insertFibEntry(client: GraphQLClient, name: string, nexthop: string): Promise<string> {
  const { insertFibEntry: { id } } = await client.request<{
    insertFibEntry: { id: string };
  }>(gql`
    mutation insertFibEntry($name: Name!, $nexthops: [ID!]!) {
      insertFibEntry(name: $name, nexthops: $nexthops) {
        id
      }
    }
  `, {
    name,
    nexthops: [nexthop],
  });
  return id;
}

async function updateNdt(client: GraphQLClient, name: string, value: number): Promise<number> {
  const { updateNdt: { index } } = await client.request<{
    updateNdt: { index: number };
  }>(gql`
    mutation updateNdt($name: Name!, $value: Int!) {
      updateNdt(name: $name, value: $value) {
        index
      }
    }
  `, {
    name,
    value,
  });
  return index;
}

function sumThroughput(a: Iterable<Throughput>): Throughput {
  const r = { pps: 0, bps: 0 };
  for (const { pps, bps } of a) {
    r.pps += pps;
    r.bps += bps;
  }
  return r;
}
