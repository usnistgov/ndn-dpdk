import numd from "numd";
import { h } from "preact";

import { gql, gqlQuery } from "./client";
import { FwFwd } from "./fw-fwd";
import { FwInput } from "./fw-input";
import { Face, Worker } from "./model";
import { TimerRefreshComponent } from "./refresh-component";
import { WorkerShape } from "./worker-shape";

interface WorkerTX extends Worker<"TX"> {
  txLoopFaces?: Array<Pick<Face, "id" | "nid">>;
}

interface FwdpQueryResult {
  fwdp: {
    inputs: Array<{
      id: string;
      nid: number;
      worker: Worker;
    }>;
    fwds: Array<{
      id: string;
      nid: number;
      worker: Worker;
    }>;
  };
  workersTX: WorkerTX[];
}

interface State {
  fwdp?: FwdpQueryResult;
}

export class FwDiagram extends TimerRefreshComponent<{}, State> {
  protected override async refresh() {
    const fwdp = await gqlQuery<FwdpQueryResult>(gql`
      {
        fwdp {
          inputs { id nid worker { ${Worker.subselection} } }
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
    const { fwdp: { inputs, fwds }, workersTX } = this.state.fwdp;
    const height = Math.max(100 * inputs.length, 120 * fwds.length, 100 * workersTX.length);
    return (
      <svg style="background: #ffffff; width: 900px;" viewBox={`0 0 900 ${height}`}>
        {fwds.flatMap((fwd, i) => [
          ...inputs.map((input, j) => (
            <line key={`${i}L${j}`} x1={200} y1={100 * j + 40} x2={300} y2={120 * i + 50} stroke="#aaaaaa" stroke-width="1"/>
          )),
          ...workersTX.map((worker, j) => (
            <line key={`${i}R${j}`} x1={700} y1={100 * j + 40} x2={600} y2={120 * i + 50} stroke="#aaaaaa" stroke-width="1"/>
          )),
        ])}
        {inputs.map(({ id, worker }, i) => (
          <WorkerShape key={id} role={worker.role} label={`input ${worker.nid}`} x={0} y={100 * i} width={200} height={80}>
            <FwInput id={id}/>
          </WorkerShape>
        ))}
        {fwds.map(({ id, worker }, i) => (
          <WorkerShape key={id} role={worker.role} label={`fwd ${worker.nid}`} x={300} y={120 * i} width={300} height={100}>
            <FwFwd id={id}/>
          </WorkerShape>
        ))}
        {workersTX.map((worker, i) => (
          <WorkerShape key={worker.id} role={worker.role} label={`output ${worker.nid}`} x={700} y={100 * i} width={200} height={80}>
            <text x="1" y="40" dominant-baseline="hanging">
              {numd(worker.txLoopFaces?.length ?? 0, "face", "faces")}
              <title>{worker.txLoopFaces?.map(({ nid }) => nid).join(", ")}</title>
            </text>
          </WorkerShape>
        ))}
      </svg>
    );
  }
}
