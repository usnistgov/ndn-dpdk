import { h } from "preact";

import { gql, gqlSub } from "./client";
import { AbortableComponent } from "./refresh-component";

interface Props {
  id: string;
}

interface State {
  itemsPerPoll: number;
  emptyPolls: number;
  validPolls: number;
}

export class ThreadLoadStat extends AbortableComponent<Props, State> {
  state: State = {
    itemsPerPoll: 0,
    emptyPolls: 0,
    validPolls: 0,
  };

  override async componentDidMount() {
    const { id } = this.props;
    for await (const { threadLoadStat } of gqlSub<{ threadLoadStat: State }>(gql`
      subscription threadLoadStat($id: ID!) {
        threadLoadStat(id: $id, interval: "1s", diff: true) {
          itemsPerPoll
          emptyPolls
          validPolls
        }
      }
    `, { id }, this.abort)) {
      this.setState(threadLoadStat);
    }
  }

  override render() {
    return (
      <span>{this.state.itemsPerPoll.toFixed(1)} × {this.state.validPolls} / {this.state.emptyPolls}</span>
    );
  }
}
