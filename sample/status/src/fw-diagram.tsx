import numd from "numd";
import { h } from "preact";

import { client, gql } from "./client";
import { type FwDispatchCounters, FwDispatchQueues } from "./fw-dispatch-queues";
import { type FwdPktQueueCounters, FwFwd } from "./fw-fwd";
import { FwInput } from "./fw-input";
import { type Face, Worker } from "./model";
import { TimerRefreshComponent } from "./refresh-component";
import { WorkerShape } from "./worker-shape";

interface WorkerTX extends Worker<"TX"> {
  txLoopFaces?: Array<Pick<Face, "id" | "nid">>;
}

interface FwdpThread {
  id: string;
  nid: number;
  worker: Worker;
}

interface FwdpQueryResult {
  fwdp: {
    inputs: FwdpThread[];
    cryptos: FwdpThread[];
    disks: FwdpThread[];
    fwds: FwdpThread[];
  };
  workersTX: WorkerTX[];
}

interface State {
  fwdp?: FwdpQueryResult;
  fwdPktQueueCounters: Array<Array<FwdPktQueueCounters | undefined> | undefined>;
  dfHighlight?: [role: string, nid: number];
}

export class FwDiagram extends TimerRefreshComponent<{}, State> {
  state: State = {
    fwdPktQueueCounters: [],
  };

  protected override async refresh() {
    const fwdp = await client.request<FwdpQueryResult>(gql`
      {
        fwdp {
          inputs { id nid worker { ${Worker.subselection} } }
          cryptos { id nid worker { ${Worker.subselection} } }
          disks { id nid worker { ${Worker.subselection} } }
          fwds { id nid worker { ${Worker.subselection} } }
        }
        workersTX: workers(role: "TX") { txLoopFaces { id nid } ${Worker.subselection} }
      }
    `);
    return { fwdp };
  }

  override render() {
    if (!this.state.fwdp) {
      return undefined;
    }
    const { fwdp: { fwdp: { inputs, cryptos, disks, fwds }, workersTX }, dfHighlight } = this.state;
    const height = 120 * Math.max(inputs.length + cryptos.length + disks.length, fwds.length, workersTX.length);
    return (
      <svg style="background: #ffffff; width: 900px;" viewBox={`0 0 900 ${height}`}>
        {fwds.flatMap(({ nid: fwd }) => [
          ...[...inputs, ...cryptos, ...disks].map(({ nid: d, worker: { role } }) => (
            <line
              key={`l F${fwd} D${d}`} x1={200} y1={120 * d + 50} x2={300} y2={120 * fwd + 50}
              stroke={
                dfHighlight === undefined ? "#aaaaaa" :
                (dfHighlight[0] === "FWD" && dfHighlight[1] === fwd) || (dfHighlight[0] === role && dfHighlight[1] === d) ? "#f012be" : "#dddddd"
              } stroke-width="1"
            />
          )),
          ...workersTX.map((worker, j) => (
            <line key={`l F${fwd} TX${j}`} x1={700} y1={120 * j + 50} x2={600} y2={120 * fwd + 50} stroke="#aaaaaa" stroke-width="1"/>
          )),
        ])}
        {[...inputs, ...cryptos, ...disks, ...fwds].map(({ nid, worker: { role } }) => {
          const x = role === "FWD" ? 300 : 200;
          const y = 120 * nid + 50;
          const arcSweep = role === "FWD" ? 0 : 1;
          return (
            <path
              key={`h${role}${nid}`} fill="#f012be"
              d={`M ${x} ${y - 10} a 10 10 180 0 ${arcSweep} 0 20 z`}
              onClick={this.handleHighlightDispatch.bind(this, role, nid)}
            >
              <title>show only packets {role === "FWD" ? "to" : "from"} this thread</title>
            </path>
          );
        })}
        {inputs.map(({ id, nid, worker }) => (
          <WorkerShape key={id} role={worker.role} label={`input ${worker.nid}`} x={0} y={120 * nid} width={200} height={100}>
            <FwInput id={id}/>
            <FwDispatchQueues
              id={id} x={200} y={15}
              onlyToFwd={dfHighlight?.[0] === "FWD" ? dfHighlight[1] : undefined}
              onChange={this.handleDispatchCntChange.bind(this, nid)}
            />
          </WorkerShape>
        ))}
        {[...cryptos, ...disks].map(({ id, nid, worker }) => (
          <WorkerShape key={id} role={worker.role} label={`${worker.role.toLowerCase()} ${worker.nid}`} x={20} y={120 * nid} width={180} height={100}>
            <FwDispatchQueues
              id={id} x={180} y={15}
              onlyToFwd={dfHighlight?.[0] === "FWD" ? dfHighlight[1] : undefined}
              onChange={this.handleDispatchCntChange.bind(this, nid)}
            />
          </WorkerShape>
        ))}
        {fwds.map(({ id, nid, worker }) => (
          <WorkerShape key={id} role={worker.role} label={`fwd ${worker.nid}`} x={300} y={120 * nid} width={300} height={100}>
            <FwFwd
              id={id}
              inputCnt={this.state.fwdPktQueueCounters?.[nid]}
              onlyFromInput={dfHighlight?.[0] === "FWD" ? undefined : dfHighlight?.[1]}
            />
          </WorkerShape>
        ))}
        {workersTX.map((worker, i) => (
          <WorkerShape key={worker.id} role={worker.role} label={`output ${worker.nid}`} x={700} y={120 * i} width={200} height={100}>
            <text x="1" y="40" dominant-baseline="hanging">
              {numd(worker.txLoopFaces?.length ?? 0, "face", "faces")}
              <title>{worker.txLoopFaces?.map(({ nid }) => nid).join(", ")}</title>
            </text>
          </WorkerShape>
        ))}
      </svg>
    );
  }

  private handleHighlightDispatch(role: string, nid: number) {
    const { dfHighlight } = this.state;
    if (dfHighlight?.[0] === role && dfHighlight?.[1] === nid) {
      this.setState({ dfHighlight: undefined });
    } else {
      this.setState({ dfHighlight: [role, nid] });
    }
  }

  private handleDispatchCntChange(d: number, cnt: FwDispatchCounters) {
    const { fwdPktQueueCounters } = this.state;
    for (const [key, a = []] of Object.entries(cnt) as Iterable<[keyof FwdPktQueueCounters, number[] | undefined]>) {
      for (const [f, n] of a.entries()) {
        ((fwdPktQueueCounters[f] ??= [])[d] ??= {})[key] = n;
      }
    }
    this.setState({ fwdPktQueueCounters });
  }
}
