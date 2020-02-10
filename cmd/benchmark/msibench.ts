import Debug = require("debug");
import * as yargs from "yargs";

import { Nanoseconds } from "../../core/nnduration/mod.js";

import * as msi from "./msi.js";
import { ITrafficGen, NdnpingTrafficGen } from "./trafficgen.js";
import { UcBenchmark } from "./ucbench.js"
import { Uncertainty } from "./uncertainty.js";

const debug = Debug("msibench");

export class MsiBenchmark extends UcBenchmark<msi.MeasureResult> {
  public msiOpts: Partial<msi.Options>;

  private intervalNearby: Nanoseconds;

  constructor(private readonly gen: ITrafficGen, uncertainty: Uncertainty) {
    super(uncertainty);
    this.msiOpts = {};
    this.intervalNearby = 0;
  }

  /**
   * Enable Executing MSI measurements in hint mode.
   *
   * When enabled, after an initial MSI observation, subsequent MSI observations
   * use the mean of previous observations as a hint, and assume the MSI is in range
   * [mean-nearby, mean+nearby]. If this hint is correct, the subsequent observation
   * would require fewer iterations and thus run faster. If the hint is incorrect,
   * hint mode will be disabled for the remaider of the benchmark.
   */
  public enableHint(nearby: Nanoseconds) {
    this.intervalNearby = nearby;
  }

  public disableHint() {
    this.enableHint(0);
  }

  protected async observe(): Promise<[msi.MeasureResult, number]> {
    const { mean } = this.uncertainty.getState();
    if (this.intervalNearby !== 0 && !isNaN(mean)) {
      const hint = Math.round(mean);
      const msiOpts = {
        ...this.msiOpts,
        IntervalMin: Math.max(hint - this.intervalNearby, 0),
        IntervalMax: hint + this.intervalNearby,
      };
      debug("applying hint %d ± %d", hint, this.intervalNearby);
      const res = await msi.measure(this.gen, msiOpts);
      if (!res.isUnderflow && !res.isOverflow) {
        return [res, res.MSI!];
      }
      debug("observe with hint failed, disable hint and retry");
      this.disableHint();
    }

    const res = await msi.measure(this.gen, this.msiOpts);
    return [res, res.isUnderflow || res.isOverflow ? NaN : res.MSI!];
  }
}

interface Argv extends msi.Options {
  DesiredUncertainty: number;
  IntervalNearby: Nanoseconds;
}

async function main() {
  const argv = yargs.parse() as Partial<Argv>;
  const gen = await NdnpingTrafficGen.create();
  const uncertainty = new Uncertainty(argv.DesiredUncertainty ?? 10);

  const mb = new MsiBenchmark(gen, uncertainty);
  mb.msiOpts = argv;
  mb.enableHint(argv.IntervalNearby ?? 500);

  mb.on("oberror", () => {
    throw new Error("MSI underflow or overflow");
  });
  mb.on("progress", (msiResult) => {
    process.stdout.write(JSON.stringify(msiResult) + "\n");
  });
  mb.on("done", (ucState) => {
    process.stdout.write(JSON.stringify(ucState) + "\n");
  });

  await mb.run();
}

if (require.main === module) {
  main()
  .catch((err) => { process.stderr.write(`${err}\n`); process.exit(1); });
}
