import { Component, h } from "preact";

import { Benchmark, BenchmarkOptions, ServerEnv } from "./benchmark";
import { BenchmarkOptionsEditor } from "./benchmark-options-editor";
import { ResultRecord, ResultTable } from "./result-table";
import { TopologyView } from "./topology-view";

interface State {
  env?: ServerEnv;
  message: string;
  opts: BenchmarkOptions;
  running: boolean;
  results: ResultRecord[];
}

export class App extends Component<{}, State> {
  state: State = {
    message: "",
    opts: {
      faceScheme: "ether",
      faceRxQueues: 1,
      nFwds: 4,
      interestNameLen: 3,
      dataMatch: "exact",
      payloadLen: 1000,
      duration: 30,
    },
    running: false,
    results: [],
  };

  private abort?: AbortController;

  override async componentDidMount() {
    const env = await (await fetch("/env.json")).json();
    this.setState({ env });
  }

  override render() {
    const { env, message, opts, running, results } = this.state;
    if (!env) {
      return <p>loading</p>;
    }
    return (
      <section>
        <TopologyView env={env}/>
        <form class="pure-form pure-form-aligned">
          <BenchmarkOptionsEditor opts={opts} disabled={running} onChange={this.handleOptsChange}>
            <div class="pure-controls">
              <button type="button" class="pure-button pure-button-primary" hidden={running} onClick={this.handleStart}>START</button>
              <button type="button" class="pure-button stop-button" hidden={!running} onClick={this.handleStop}>STOP</button>
            </div>
          </BenchmarkOptionsEditor>
        </form>
        <p><code>{message}</code></p>
        <ResultTable records={results}/>
      </section>
    );
  }

  private readonly handleOptsChange = (update: Partial<BenchmarkOptions>) => {
    this.setState(({ opts }) => {
      opts = { ...opts, ...update };
      return { opts };
    });
  };

  private readonly handleStart = () => {
    this.setState(
      ({ env, opts }) => {
        const fDemand = 2 * opts.faceRxQueues + 2 + opts.nFwds;
        const gDemand = 2 * opts.faceRxQueues + 2 + 2 + 2;
        if (fDemand > env!.F_CORES_PRIMARY.length || gDemand > env!.G_CORES_PRIMARY.length) {
          return {
            message: `insufficient CPU cores: need ${fDemand} on forwarder and ${gDemand} on traffic gen`,
          };
        }

        this.abort = new AbortController();
        return { running: true, results: [] };
      },
      this.run,
    );
  };

  private readonly run = async () => {
    if (!this.state.running) {
      return;
    }
    const abort = this.abort!;
    try {
      const b = new Benchmark(this.state.env!, this.state.opts, abort.signal);
      this.setState({ message: "starting forwarder and traffic generator" });
      await Promise.all([
        b.setupForwarder(),
        b.setupTrafficGen(),
      ]);

      let i = 0;
      while (this.state.running) {
        this.setState({ message: `running trial ${++i}` });
        const r = await b.run();
        const dt = new Date();
        const { pps, bps } = b.computeThroughput(r);
        console.log({ i, dt, pps, bps, r });
        this.setState(({ results }) => ({
          results: [...results, { dt, pps, bps }],
        }));
      }
    } catch (err: unknown) {
      console.error(err);
      if (this.abort === abort) {
        this.setState({ running: false, message: `${err}` });
      }
    }
  };

  private readonly handleStop = () => {
    this.setState(
      () => ({ running: false, message: "stopping" }),
      () => {
        this.abort?.abort();
        this.abort = undefined;
      },
    );
  };
}
