import type { Counter } from "@usnistgov/ndn-dpdk";
import { Fragment, h } from "preact";

import { gql, GqlErrors, gqlSubError } from "./client";
import { AbortableComponent } from "./refresh-component";

interface Props {
  id: string;
}

interface LoadStat {
  itemsPerPoll: number;
  emptyPolls: Counter;
  validPolls: Counter;
}

interface State {
  loadStat?: LoadStat;
}

export class WorkerLoadStat extends AbortableComponent<Props, State> {
  state: State = {};

  override async componentDidMount() {
    const { id } = this.props;
    for await (const item of gqlSubError<{ threadLoadStat: LoadStat }>(gql`
      subscription threadLoadStat($id: ID!) {
        threadLoadStat(id: $id, interval: "1s", diff: true) {
          itemsPerPoll
          emptyPolls
          validPolls
        }
      }
    `, { id }, this.abort)) {
      if (item instanceof GqlErrors) {
        this.setState({ loadStat: undefined });
      } else {
        this.setState({ loadStat: item.threadLoadStat });
      }
    }
  }

  override render() {
    const { loadStat } = this.state;
    if (!loadStat) {
      return (
        <>
          <td colSpan={5}/>
          <td title="Thread is either stopped or cannot report load statistics.">🟡</td>
        </>
      );
    }

    const { itemsPerPoll, validPolls, emptyPolls } = loadStat;
    const warnOverloaded = emptyPolls < validPolls;
    return (
      <>
        <td style="text-align: right;">{itemsPerPoll.toFixed(1)}</td>
        <td>×</td>
        <td style="text-align: right;">{validPolls}</td>
        <td>/</td>
        <td style="text-align: right;">{emptyPolls}</td>
        <td title={warnOverloaded ? "Thread is possibly overloaded." : ""}>{warnOverloaded ? "🔴" : "🟢"}</td>
      </>
    );
  }
}
