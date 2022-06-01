import type { Counter, TgpCounters, TgpPattern, TgpReply } from "@usnistgov/ndn-dpdk";
import { type ComponentChild, Fragment, h } from "preact";

import { gql, gqlSub } from "./client";
import { formatName } from "./model";
import { AbortableComponent } from "./refresh-component";

interface Props {
  tgID: string;
  patterns: TgpPattern[];
}

interface State {
  cnt?: TgpCounters;
}

export class TgpPatternsTable extends AbortableComponent<Props, State> {
  state: State = {};

  override async componentDidMount() {
    const { tgID } = this.props;
    for await (const { tgCounters: { producer: cnt } } of gqlSub<{ tgCounters: { producer: TgpCounters } }>(gql`
      subscription tgCounters($id: ID!) {
        tgCounters(id: $id, interval: "1s", diff: true) {
          producer {
            nAllocError nInterests nNoMatch
            perPattern { nInterests perReply }
          }
        }
      }
    `, { id: tgID }, this.abort)) {
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
            <th>reply</th>
            <th>Interest/s</th>
          </tr>
        </thead>
        <tbody>
          {patterns.map((pattern, i) => this.renderPattern(i, pattern, cnt?.perPattern[i]))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={2}>
              <span title={`${cnt?.nAllocError} alloc error`}>{cnt?.nAllocError === 0 ? "🟢" : "🔴"}</span>
              <span title={`${cnt?.nNoMatch} unmatched`}>{cnt?.nNoMatch === 0 ? "🟢" : "🔴"}</span>
            </td>
            <td style="text-align: right;">{cnt?.nInterests}</td>
            <td/>
            <td/>
          </tr>
        </tfoot>
      </table>
    );
  }

  private renderPattern(i: number, pattern: TgpPattern, cnt: Partial<TgpCounters.PatternCounters> = {}) {
    return pattern.replies.map((reply, j, { length: nReplies }) => (
      this.renderReply(i, j, reply, cnt.perReply?.[j], j === 0 ? (
        <>
          <td rowSpan={nReplies}>{i}</td>
          <td rowSpan={nReplies}>{formatName(pattern.prefix)}</td>
          <td rowSpan={nReplies} style="text-align: right;">{cnt.nInterests}</td>
        </>
      ) : undefined)
    ));
  }

  private renderReply(i: number, j: number, reply: TgpReply, cnt?: Counter, children?: ComponentChild) {
    return (
      <tr key={`${i}.${j}`}>
        {children}
        <td>{describeReply(reply)}</td>
        <td style="text-align: right;">{cnt}</td>
      </tr>
    );
  }
}

function describeReply(reply: TgpReply) {
  if ((reply as TgpReply.Timeout).timeout) {
    return "drop";
  }

  const { nack } = reply as TgpReply.Nack;
  if (nack) {
    return `Nack~${nack}`;
  }

  const {
    suffix,
    payloadLen = 0,
  } = reply as TgpReply.Data;
  return `Data ${suffix ? `append(${suffix.match(/\//g)?.length})` : "exact"} ${payloadLen}B`;
}
