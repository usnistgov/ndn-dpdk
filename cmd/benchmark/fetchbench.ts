import * as _ from "lodash";
import * as yargs from "yargs";

import * as mgmt from "../../mgmt/mod.js";
import { Name } from "../../ndn/mod.js";
import { FetchIndexArg, FetchBenchmarkArgs, FetchBenchmarkReply } from "../../mgmt/pingmgmt/mod.js";

import { UcBenchmark } from "./ucbench.js"
import { Uncertainty } from "./uncertainty.js";

interface NameGenArgs {
  NameTemplate: string;
  NameCount: number;
}

type BenchArgs = Omit<FetchBenchmarkArgs, "Names"|keyof FetchIndexArg> & NameGenArgs & {
  NameGen: { [k: string]: Partial<NameGenArgs> };
};

interface ObserveResultEntry {
  Names: Name[];
  Result: FetchBenchmarkReply;
}

export class FetchBenchmark extends UcBenchmark<ObserveResultEntry[]> {
  private rpc: mgmt.RpcClient;
  private opts: BenchArgs;
  private fetchers: FetchIndexArg[] = [];

  constructor(uncertainty: Uncertainty, opts: Partial<BenchArgs> = {}) {
    super(uncertainty);
    this.rpc = mgmt.makeMgmtClient();

    this.opts = {
      NameGen: {},
      NameTemplate: "/8=fetch/8=#/8=*",
      NameCount: 1,
      Warmup: 5000,
      Interval: 10,
      Count: 20000,
      ...opts,
    };
  }

  protected async prepare() {
    this.fetchers = await this.rpc.request<{}, FetchIndexArg[]>("Fetch.List", {});
  }

  protected async observe(): Promise<[ObserveResultEntry[], number]> {
    const prefixRand = Math.floor(Math.random() * 99999999);
    const results = await Promise.all(this.fetchers.map(async (fetchIndex) => {
      const Names = this.makeNames(prefixRand, fetchIndex);
      const Result = await this.rpc.request<FetchBenchmarkArgs, FetchBenchmarkReply>("Fetch.Benchmark",
                           { ...fetchIndex, ...this.opts, Names });
      return { Names, Result } as ObserveResultEntry;
    }));
    const totalGoodput = _.sumBy(results, (r) => r.Result.Goodput);
    return [results, totalGoodput];
  }

  private makeNames(prefixRand: number, { Index, FetchId }: FetchIndexArg): Name[] {
    const fetchIndex = `${Index}-${FetchId}`;
    const { NameTemplate, NameCount } = {
      ...this.opts,
      ...this.opts.NameGen[fetchIndex],
    } as NameGenArgs;

    const prefix = NameTemplate.replace(/[*]/g, `${prefixRand}`).replace(/[@]/g, fetchIndex);
    return _.range(NameCount).map((i) => prefix.replace(/[#]/g, `${i}`));
  }
}

interface Argv extends BenchArgs {
  DesiredUncertainty: number;
}

async function main() {
  const argv = yargs.parse() as Partial<Argv>;
  const uncertainty = new Uncertainty(argv.DesiredUncertainty ?? 1000);

  const ben = new FetchBenchmark(uncertainty, argv);

  ben.on("oberror", () => {
    throw new Error("Fetch.Benchmark error");
  });
  ben.on("progress", (result) => {
    process.stdout.write(JSON.stringify(result) + "\n");
  });
  ben.on("done", (ucState) => {
    process.stdout.write(JSON.stringify(ucState) + "\n");
  });

  await ben.run();
}

if (require.main === module) {
  main()
  .catch((err) => { process.stderr.write(`${err}\n`); process.exit(1); });
}
