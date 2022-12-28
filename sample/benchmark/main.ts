import getStdin from "get-stdin";
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
  ).command("benchmark", "run benchmark on CLI (pass BenchmarkOptions to stdin)",
    (argv) => argv
      .option("count", { alias: "c", default: 1e3, desc: "iteration count", type: "number" }),
    async (argv) => {
      const opts = JSON.parse(await getStdin());
      await runBenchmark(opts, argv.count);
    },
  ).parseAsync();
