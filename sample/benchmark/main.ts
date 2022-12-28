import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import { runBenchmark } from "./benchmark";
import { serve } from "./serve";

await yargs(hideBin(process.argv))
  .scriptName("ndndpdk-benchmark")
  .command("serve", "serve webapp",
    (argv) => argv.option("port", { default: 3333, desc: "listen port", type: "number" }),
    async (argv) => {
      await serve(argv.port);
    },
  ).command("benchmark [opts]", "run benchmark on CLI",
    (argv) => argv
      .option("count", { alias: "c", default: 1e3, desc: "iteration count", type: "number" })
      .positional("opts", { coerce: (s) => JSON.parse(s), demandOption: true, type: "string" }),
    async (argv) => {
      await runBenchmark(argv.opts, argv.count);
    },
  ).parseAsync();
