import { Component, h } from "preact";

import type { BenchmarkResult } from "./benchmark";

interface Props {
  records: BenchmarkResult[];
}

const timeFmt = new Intl.DateTimeFormat([], { timeStyle: "medium" });
const floatFmt = new Intl.NumberFormat([], { maximumFractionDigits: 3 });

export class ResultTable extends Component<Props> {
  override render() {
    const { records } = this.props;
    return (
      <table class="pure-table">
        <thead>
          <tr>
            <th title="run completion time">timestamp</th>
            <th title="duration to finish file download, averaged over all parallel flows">duration</th>
            <th title="retrieved Data packets per second">Data packets throughput</th>
            <th title="retrieved Data payload bits per second">goodput</th>
          </tr>
        </thead>
        <tbody>
          {records.map(({ i, dt, duration, pps, bps }) => (
            <tr key={i}>
              <td>{timeFmt.format(dt)}</td>
              <td>{floatFmt.format(duration)} s</td>
              <td>{floatFmt.format(pps / 1e6)} Mpps</td>
              <td>{floatFmt.format(bps / 1e9)} Gbps</td>
            </tr>
          ))}
        </tbody>
      </table>
    );
  }
}
