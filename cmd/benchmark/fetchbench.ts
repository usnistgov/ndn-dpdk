import * as yargs from "yargs";

import * as mgmt from "../../mgmt/mod.js";
import { FetchBenchmarkArgs, FetchBenchmarkReply } from "../../mgmt/pingmgmt/mod.js";

import { UcBenchmark } from "./ucbench.js"
import { Uncertainty } from "./uncertainty.js";

export class FetchBenchmark extends UcBenchmark<FetchBenchmarkReply> {
  private rpc: mgmt.RpcClient;
  private opts: FetchBenchmarkArgs;
  public randomizeName = false;

  constructor(uncertainty: Uncertainty, opts: Partial<FetchBenchmarkArgs> = {}) {
    super(uncertainty);
    this.rpc = mgmt.makeMgmtClient();
    this.opts = {
      Index: 0,
      Name: "/8=fetch",
      Warmup: 5000,
      Interval: 10,
      Count: 20000,
      ...opts,
    };
  }

  protected async observe(): Promise<[FetchBenchmarkReply, number]> {
    const opts = {  ...this.opts };
    if (this.randomizeName) {
      opts.Name += `/${Math.floor(Math.random() * 99999999)}`;
    }
    const res = await this.rpc.request<FetchBenchmarkArgs, FetchBenchmarkReply>("Fetch.Benchmark", opts);
    return [res, res.Goodput];
  }
}

interface Argv extends FetchBenchmarkArgs {
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
