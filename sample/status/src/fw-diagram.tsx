import { Component, h } from "preact";

import { gql, gqlQuery } from "./client";
import { FwFwd } from "./fw-fwd";
import type { Worker } from "./model";
import { WorkerShape } from "./worker-shape";

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
  workersTX: Worker[];
}

interface State {
  fwdp?: FwdpQueryResult;
}

export class FwDiagram extends Component<{}, State> {
  override async componentDidMount() {
    const fwdp = await gqlQuery<FwdpQueryResult>(gql`
      {
        fwdp {
          inputs { id nid worker { id nid numaSocket role } }
          fwds { id nid worker { id nid numaSocket role } }
        }
        workersTX: workers(role: "TX") { id nid numaSocket role }
      }
    `);
    this.setState({
      fwdp,
    });
  }

  override render() {
    if (!this.state.fwdp) {
      return undefined;
    }
    const { fwdp: { inputs, fwds }, workersTX } = this.state.fwdp;
    const height = Math.max(100 * inputs.length, 120 * fwds.length, 100 * workersTX.length);
    return (
      <svg style="background: #ffffff; width: 900px;" viewBox={`0 0 900 ${height}`}>
        {inputs.map(({ id, worker }, i) => (
          <WorkerShape key={id} role={worker.role} label={`input ${worker.nid}`} x={0} y={100 * i} width={200} height={80}/>
        ))}
        {fwds.map(({ id, worker }, i) => (
          <WorkerShape key={id} role={worker.role} label={`fwd ${worker.nid}`} x={300} y={120 * i} width={300} height={100}>
            <FwFwd id={id}/>
          </WorkerShape>
        ))}
        {workersTX.map((worker, i) => (
          <WorkerShape key={worker.id} role={worker.role} label={`output ${worker.nid}`} x={700} y={100 * i} width={200} height={80}/>
        ))}
      </svg>
    );
  }
}
