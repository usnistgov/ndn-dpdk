import * as _ from "lodash";
import * as yargs from "yargs";

import * as mgmt from "../../mgmt/mod.js";
import { Name } from "../../ndn/mod.js";
import { FetchBenchmarkArgs, FetchBenchmarkReply } from "../../mgmt/pingmgmt/mod.js";

import { UcBenchmark } from "./ucbench.js"
import { Uncertainty } from "./uncertainty.js";

interface NameGenArgs {
  NamePrefix: Name;
  NameCount: number;
}

type BenchArgs = Omit<FetchBenchmarkArgs, "Names"> & NameGenArgs;

export class FetchBenchmark extends UcBenchmark<FetchBenchmarkReply> {
  private rpc: mgmt.RpcClient;
  private opts: BenchArgs;
  public randomizeName = false;

  constructor(uncertainty: Uncertainty, opts: Partial<BenchArgs> = {}) {
    super(uncertainty);
    this.rpc = mgmt.makeMgmtClient();

    this.opts = {
      Index: 0,
      Warmup: 5000,
      Interval: 10,
      Count: 20000,

      NamePrefix: "/8=fetch",
      NameCount: 1,

      ...opts,
    };
  }

  protected async observe(): Promise<[FetchBenchmarkReply, number]> {
    const opts = {
      ...this.opts,
      Names: this.makeNames(),
    };
    const res = await this.rpc.request<FetchBenchmarkArgs, FetchBenchmarkReply>("Fetch.Benchmark", opts);
    return [res, res.Goodput];
  }

  private makeNames(): Name[] {
    const suffix = this.randomizeName ? `/8=${Math.floor(Math.random() * 99999999)}` : "";
    return _.range(this.opts.NameCount).map((i) => `${this.opts.NamePrefix}/8=${i}${suffix}`);
  }
}

interface Argv extends BenchArgs {
  DesiredUncertainty: number;
}

async function main() {
  const argv = yargs.parse() as Partial<Argv>;
  const uncertainty = new Uncertainty(argv.DesiredUncertainty ?? 1000);

  const ben = new FetchBenchmark(uncertainty, argv);
  ben.randomizeName = true;

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
