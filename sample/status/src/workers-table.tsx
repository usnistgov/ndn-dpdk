import { Component, h } from "preact";

import type { WorkersByRole } from "./model";
import { WorkerLoadStat } from "./worker-load-stat";

interface Props {
  workers: WorkersByRole;
}

export class WorkersTable extends Component<Props> {
  override render() {
    const workerEntries = Object.entries(this.props.workers).sort(([a], [b]) => a.localeCompare(b));
    return (
      <table class="pure-table pure-table-horizontal">
        <thead>
          <tr>
            <th rowSpan={2}>role</th>
            <th rowSpan={2}>#</th>
            <th rowSpan={2}>NUMA</th>
            <th colSpan={6}>load</th>
          </tr>
          <tr>
            <th colSpan={2}>items per poll</th>
            <th colSpan={2}>valid polls</th>
            <th colSpan={2}>empty polls</th>
          </tr>
        </thead>
        <tbody>
          {workerEntries.map(([role, workers]) => workers.sort((a, b) => a.nid - b.nid).map((w, i) => (
            <tr key={w.id}>
              {i === 0 ? (
                <td rowSpan={workers.length}>{role}</td>
              ) : undefined}
              <td title={w.id}>{w.nid}</td>
              <td>{w.numaSocket}</td>
              <WorkerLoadStat id={w.id}/>
            </tr>
          )))}
        </tbody>
      </table>
    );
  }
}
