import { type TgcCounters, type TgcPattern, runningstat } from "@usnistgov/ndn-dpdk";
import { Fragment, h } from "preact";

import { client, gql } from "./client";
import { formatName } from "./model";
import { AbortableComponent } from "./refresh-component";

interface Props {
  tgID: string;
  patterns: TgcPattern[];
}

interface State {
  cnt?: TgcCounters;
}

export class TgcPatternsTable extends AbortableComponent<Props, State> {
  state: State = {};

  override async componentDidMount() {
    const { tgID } = this.props;
    for await (const { consumer: cnt } of client.subscribe<{ consumer: TgcCounters }>(gql`
      subscription tgCounters($id: ID!) {
        tgCounters(id: $id, interval: "1s", diff: true) {
          consumer {
            nAllocError nInterests nData nNacks
            perPattern {
              nInterests nData nNacks
              rtt { count len m1 m2 }
            }
          }
        }
      }
    `, { id: tgID }, { signal: this.signal, key: "tgCounters" })) {
      this.setState({ cnt });
    }
  }

  override render() {
    const { patterns } = this.props;
    const { cnt } = this.state;
    return (
      <table class="pure-table pure-table-horizontal">
        <thead>
          <tr>
            <th>#</th>
            <th>pattern</th>
            <th>Interest/s</th>
            <th>Data/s</th>
            <th>Data %</th>
            <th>RTT (μs)</th>
            <th>RTT (σ)</th>
            <th>Nack/s</th>
            <th>Nack %</th>
          </tr>
        </thead>
        <tbody>
          {patterns.map((pattern, i) => this.renderPattern(i, pattern, cnt?.perPattern[i]))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={2}>
              <span title={`${cnt?.nAllocError} alloc error`}>{cnt?.nAllocError === 0 ? "🟢" : "🔴"}</span>
            </td>
            {this.renderCounters(cnt)}
          </tr>
        </tfoot>
      </table>
    );
  }

  private renderPattern(i: number, pattern: TgcPattern, cnt: Partial<TgcCounters.PatternCounters> = {}) {
    const rtt = new runningstat.Snapshot(cnt.rtt).scale(0.001);
    return (
      <tr key={i}>
        <td>{i}</td>
        <td title={JSON.stringify(pattern, undefined, 2)}>{describePattern(pattern)}</td>
        {this.renderCounters(cnt, rtt)}
      </tr>
    );
  }

  private renderCounters(cnt: Partial<TgcCounters.PacketCounters> = {}, rtt?: runningstat.Snapshot) {
    const { nInterests = 0, nData = 0, nNacks = 0 } = cnt;
    const dataPct = (Number(nData) / Number(nInterests) * 100).toFixed(2);
    const nackPct = (Number(nNacks) / Number(nInterests) * 100).toFixed(2);
    return (
      <>
        <td style="text-align: right;">{nInterests}</td>
        <td style="text-align: right;">{nData}</td>
        <td style="text-align: right;">{dataPct}</td>
        <td style="text-align: right;">{rtt?.mean.toFixed(0)}</td>
        <td style="text-align: right;">{rtt?.stdev.toFixed(0)}</td>
        <td style="text-align: right;">{nNacks}</td>
        <td style="text-align: right;">{nackPct}</td>
      </>
    );
  }
}

function describePattern({
  seqNumOffset = 0,
  digest,
  prefix,
  canBePrefix = false,
  mustBeFresh = false,
}: TgcPattern) {
  return (
    <>
      <strong>{formatName(prefix)}</strong>
      {seqNumOffset === 0 ? (<em>/seq</em>) : (<em>/{-seqNumOffset}</em>)}
      {digest && (<em>/digest</em>)}
      {canBePrefix && (<small> [P]</small>)}
      {mustBeFresh && (<small> [F]</small>)}
    </>
  );
}
