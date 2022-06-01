import type { Counter, FileServerCounters } from "@usnistgov/ndn-dpdk";
import { h } from "preact";

import { gql, gqlSub } from "./client";
import { AbortableComponent } from "./refresh-component";

interface Props {
  tgID: string;
}

const counters = [
  "reqRead", "reqLs", "reqMetadata",
  "fdNew", "fdNotFound", "fdUpdateStat", "fdClose",
  "uringSubmit", "uringSubmitNonBlock", "uringSubmitWait", "sqeSubmit", "cqeFail",
] as const;

interface State {
  cnt: FileServerCounters;
}

export class FileServerCounterView extends AbortableComponent<Props, State> {
  state: State = {
    cnt: Object.fromEntries(counters.map((cnt) => [cnt, 0])) as any,
  };

  override async componentDidMount() {
    const { tgID } = this.props;
    for await (const { tgCounters: { fileServer: cnt } } of gqlSub<{ tgCounters: { fileServer: FileServerCounters } }>(gql`
      subscription tgCounters($id: ID!) {
        tgCounters(id: $id, interval: "1s", diff: false) {
          fileServer { ${counters.join(" ")} }
        }
      }
    `, { id: tgID }, this.abort)) {
      this.setState({ cnt });
    }
  }

  override render() {
    const { cnt } = this.state;
    return (
      <div style="display: flex; flex-direction: row; flex-wrap: wrap;">
        <table class="pure-table pure-table-horizontal" style="margin: 0 1em;">
          <caption>requests</caption>
          {this.renderCounter(cnt.reqRead, "read")}
          {this.renderCounter(cnt.reqLs, "ls")}
          {this.renderCounter(cnt.reqMetadata, "metadata")}
        </table>
        <table class="pure-table pure-table-horizontal" style="margin: 0 1em;">
          <caption>file descriptors</caption>
          {this.renderCounter(cnt.fdNew, "new")}
          {this.renderCounter(cnt.fdNotFound, "not found", "Consumers are requesting non-existent files.")}
          {this.renderCounter(cnt.fdUpdateStat, "update stat")}
          {this.renderCounter(cnt.fdClose, "close")}
        </table>
        <table class="pure-table pure-table-horizontal" style="margin: 0 1em;">
          <caption>io_uring</caption>
          {this.renderCounter(cnt.uringSubmit, "submit bursts")}
          {this.renderCounter(cnt.uringSubmitNonBlock, "(non-blocking)")}
          {this.renderCounter(cnt.uringSubmitWait, "(with wait)", "Some I/O requests are submitted with waiting for completions due to insufficient I/O bandwidth.")}
          {this.renderCounter(cnt.sqeSubmit, "submit SQEs")}
          {this.renderCounter(cnt.cqeFail, "failed CQEs", "Some I/O requests are failing.")}
        </table>
      </div>
    );
  }

  private renderCounter(value: Counter, title: string, positiveWarning?: string) {
    return (
      <tr>
        <td>{title}</td>
        <td style="text-align: right;">{value}</td>
        {positiveWarning === undefined ? <td/> : (
          Number(value) > 0 ? <td title={positiveWarning}>🔴</td> : <td>🟢</td>
        )}
      </tr>
    );
  }
}
