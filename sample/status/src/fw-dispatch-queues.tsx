import { h } from "preact";

import { gql, gqlSub } from "./client";
import { AbortableComponent } from "./refresh-component";

const counters = [
  "nInterestsQueued", "nInterestsDropped",
  "nDataQueued", "nDataDropped",
  "nNacksQueued", "nNacksDropped",
] as const;

export interface FwDispatchCounters extends Partial<Record<typeof counters[number], number[]>> {
}

interface Props {
  id: string;
  x: number;
  y: number;
  onlyToFwd?: number;
  onChange?: (cnt: FwDispatchCounters) => void;
}

interface State {
  cnt: FwDispatchCounters;
}

export class FwDispatchQueues extends AbortableComponent<Props, State> {
  state: State = {
    cnt: {},
  };

  override async componentDidMount() {
    const { id, onChange } = this.props;
    for await (const { fwDispatchCounters } of gqlSub<{ fwDispatchCounters: FwDispatchCounters }>(gql`
      subscription fwDispatchCounters($id: ID!) {
        fwDispatchCounters(id: $id, interval: "1s", diff: true) {
          ${counters.join(" ")}
        }
      }
    `, { id }, this.abort)) {
      this.setState({ cnt: fwDispatchCounters });
      onChange?.(fwDispatchCounters);
    }
  }

  override render() {
    const { x, y } = this.props;
    return (
      <g transform={`translate(${x} ${y})`}>
        {this.renderQueue(0, "Interests")}
        {this.renderQueue(25, "Data")}
        {this.renderQueue(50, "Nacks")}
      </g>
    );
  }

  private renderQueue(y: number, t: "Interests" | "Data" | "Nacks") {
    const queued = this.state.cnt[`n${t}Queued`];
    const dropped = this.state.cnt[`n${t}Dropped`];
    if (!(queued?.length) || !(dropped?.length)) {
      return;
    }
    const nQueued = this.filterDest(queued);
    const nDropped = this.filterDest(dropped);
    return (
      <g transform={`translate(-98 ${y})`}>
        <rect
          width="100" height="20"
          stroke="#ff851b" stroke-width="1" fill="#ffffff"
        />
        <text x="5" y="15">{t[0]}<title>{t}</title></text>
        <text x="75" y="15" text-anchor="end">{nQueued}</text>
        <circle cx="90" cy="10" r="6" fill={nDropped === 0 ? "#2ecc40" : "#ff4136"}>
          <title>{nDropped} dropped</title>
        </circle>
      </g>
    );
  }

  private filterDest(a: readonly number[]) {
    const { onlyToFwd } = this.props;
    if (onlyToFwd === undefined) {
      return a.reduce((sum, v) => sum + v, 0);
    }
    return a[onlyToFwd] ?? 0;
  }
}
