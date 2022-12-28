import { setMaxListeners } from "node:events";

import ndjson from "ndjson";
import stdout from "stdout-stream";

import { env } from "./env";
import { type BenchmarkOptions, Benchmark } from "./src/benchmark";

setMaxListeners(150);

export async function runBenchmark(opts: BenchmarkOptions, count: number): Promise<void> {
  const output = ndjson.stringify();
  output.pipe(stdout);

  const abort = new AbortController();
  process.once("SIGINT", () => abort.abort());

  const b = new Benchmark(env, opts, abort.signal);
  output.write(opts);
  await b.setup();
  for (let i = 0; i < count && !abort.signal.aborted; ++i) {
    const result = await b.run();
    output.write(result);
  }
  abort.abort();
  output.end();
}
