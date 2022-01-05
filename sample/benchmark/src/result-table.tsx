import { Component, h } from "preact";

export interface ResultRecord {
  dt: Date;
  pps: number;
  bps: number;
}

interface Props {
  records: ResultRecord[];
}

const timeFmt = new Intl.DateTimeFormat([], { timeStyle: "medium" });
const floatFmt = new Intl.NumberFormat([], { maximumFractionDigits: 3 });

export class ResultTable extends Component<Props> {
  override render() {
    return (
      <table class="pure-table">
        <thead>
          <tr>
            <th>timestamp</th>
            <th>Data packets throughput</th>
            <th>goodput</th>
          </tr>
        </thead>
        <tbody>
          {this.props.records.map(({ dt, pps, bps }) => (
            <tr key={dt.getTime()}>
              <td>{timeFmt.format(dt)}</td>
              <td>{floatFmt.format(pps / 1e6)} Mpps</td>
              <td>{floatFmt.format(bps / 1e9)} Gbps</td>
            </tr>
          ))}
        </tbody>
      </table>
    );
  }
}
