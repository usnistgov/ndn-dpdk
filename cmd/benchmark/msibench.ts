import fail = require("@zingle/fail");
import Debug = require("debug");
import * as yargs from "yargs";

import { NdnpingTrafficGen } from "./trafficgen";

import * as msi from "./msi";
import { Uncertainty } from "./uncertainty";

const debug = Debug("msibench");

interface Argv extends Partial<msi.Options> {
  DesiredUncertainty?: number;
}

async function main() {
  const argv = yargs.parse() as Argv;

  const gen = await NdnpingTrafficGen.create();
  const uncertainty = new Uncertainty(argv.DesiredUncertainty || 10);
  while (true) {
    const ucState = uncertainty.getState();
    debug(ucState);
    if (ucState.isSufficient) {
      process.stdout.write(JSON.stringify(ucState) + "\n");
      break;
    }
    const res = await msi.measure(gen, argv);
    process.stdout.write(JSON.stringify(res) + "\n");
    if (!res.MSI || res.isUnderflow || res.isOverflow) {
      throw new Error("MSI underflow or overflow");
    }
    uncertainty.addObservation(res.MSI);
  }
}

if (require.main === module) {
  main()
  .catch(fail);
}
