import { Component, Fragment, h } from "preact";
import { mean, sampleStandardDeviation } from "simple-statistics";

import type { BenchmarkResult } from "./benchmark";

interface Props {
  records: readonly BenchmarkResult[];
  running: boolean;
}

const timeFmt = new Intl.DateTimeFormat([], { timeStyle: "medium" });
const floatFmt = new Intl.NumberFormat([], { minimumFractionDigits: 3, maximumFractionDigits: 3 });

export class ResultTable extends Component<Props> {
  override render() {
    const { records, running } = this.props;
    return (
      <table class="pure-table pure-table-horizontal">
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
          {running ? (
            <tr>
              <td>current</td>
              <td colSpan={3}>in progress</td>
            </tr>
          ) : undefined}
        </tbody>
        <tfoot>
          <tr>
            <th title="mean">average</th>
            {this.props.records.length > 0 ? (<>
              <td>{floatFmt.format(mean(this.series("duration")))} s</td>
              <td>{floatFmt.format(mean(this.series("pps")) / 1e6)} Mpps</td>
              <td>{floatFmt.format(mean(this.series("bps")) / 1e9)} Gbps</td>
            </>) : (
              <td colSpan={3}>waiting for results</td>
            )}
          </tr>
          <tr>
            <th title="sample standard deviation">stdev</th>
            {this.props.records.length >= 2 ? (<>
              <td>{floatFmt.format(sampleStandardDeviation(this.series("duration")))} s</td>
              <td>{floatFmt.format(sampleStandardDeviation(this.series("pps")) / 1e6)} Mpps</td>
              <td>{floatFmt.format(sampleStandardDeviation(this.series("bps")) / 1e9)} Gbps</td>
            </>) : (
              <td colSpan={3}>waiting for results</td>
            )}
          </tr>
        </tfoot>
      </table>
    );
  }

  private series(key: keyof BenchmarkResult): number[] {
    const { records } = this.props;
    return records.map((record) => record[key]);
  }
}
