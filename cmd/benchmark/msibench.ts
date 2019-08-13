import fail = require("@zingle/fail");
import Debug = require("debug");
import EventEmitter = require("events");
import * as yargs from "yargs";

import { NNDuration } from "../../core";

import * as msi from "./msi";
import { ITrafficGen, NdnpingTrafficGen } from "./trafficgen";
import { Uncertainty, UncertaintyState } from "./uncertainty";

const debug = Debug("msibench");

export class MsiBenchmark extends EventEmitter {
  /**
   * Event after each MSI measurement.
   * Arguments: msi.MeasureResult, UncertaintyState
   * @event
   */
  public static EVENT_PROGRESS = "progress";

  /**
   * Event upon failed MSI measurement.
   * Arguments: msi.MeasureResult
   * @event
   */
  public static EVENT_MSIERROR = "msi-error";

  /**
   * Event upon benchmark completion.
   * Arguments: UncertaintyState
   * @event
   */
  public static EVENT_DONE = "done";

  public msiOpts: Partial<msi.Options>;

  private gen: ITrafficGen;
  private uncertainty: Uncertainty;
  private intervalNearby: NNDuration;

  constructor(gen: ITrafficGen, uncertainty: Uncertainty) {
    super();
    this.gen = gen;
    this.uncertainty = uncertainty;
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
  public enableHint(nearby: NNDuration) {
    this.intervalNearby = nearby;
  }

  public disableHint() {
    this.enableHint(0);
  }

  public async run(): Promise<boolean> {
    let ucState = this.uncertainty.getState();
    while (!ucState.isSufficient) {
      const res = await this.observe();
      debug("observe result %j", res);
      if (!res.MSI || res.isUnderflow || res.isOverflow) {
        this.emit(MsiBenchmark.EVENT_MSIERROR, res);
        return false;
      }

      this.uncertainty.addObservation(res.MSI);
      ucState = this.uncertainty.getState();
      debug("ucState %j", ucState);
      this.emit(MsiBenchmark.EVENT_PROGRESS, res, ucState);
    }
    this.emit(MsiBenchmark.EVENT_DONE, ucState);
    return true;
  }

  private async observe(): Promise<msi.MeasureResult> {
    const { mean } = this.uncertainty.getState();
    if (this.intervalNearby !== 0 && !isNaN(mean)) {
      const hint = Math.round(mean);
      const msiOpts = Object.assign({}, this.msiOpts);
      msiOpts.IntervalMin = Math.max(hint - this.intervalNearby, 0);
      msiOpts.IntervalMax = hint + this.intervalNearby;
      debug("applying hint %d ± %d", hint, this.intervalNearby);
      const res1 = await msi.measure(this.gen, msiOpts);
      if (!res1.isUnderflow && !res1.isOverflow) {
        return res1;
      }
      debug("observe with hint failed, disable hint and retry");
      this.disableHint();
    }
    return await msi.measure(this.gen, this.msiOpts);
  }
}

interface Argv extends msi.Options {
  DesiredUncertainty: number;
  IntervalNearby: NNDuration;
}

async function main() {
  const argv = yargs.parse() as Partial<Argv>;
  const gen = await NdnpingTrafficGen.create();
  const uncertainty = new Uncertainty(argv.DesiredUncertainty || 10);

  const mb = new MsiBenchmark(gen, uncertainty);
  mb.msiOpts = argv;
  mb.enableHint(argv.IntervalNearby || 500);

  mb.on(MsiBenchmark.EVENT_MSIERROR, (msiResult) => {
    throw new Error("MSI underflow or overflow");
  });
  mb.on(MsiBenchmark.EVENT_PROGRESS, (msiResult: msi.MeasureResult, ucState) => {
    process.stdout.write(JSON.stringify(msiResult) + "\n");
  });
  mb.on(MsiBenchmark.EVENT_DONE, (ucState: UncertaintyState) => {
    process.stdout.write(JSON.stringify(ucState) + "\n");
  });

  await mb.run();
}

if (require.main === module) {
  main()
  .catch(fail);
}
