import type { FaceLocator } from "@usnistgov/ndn-dpdk";
import { h } from "preact";

import { gql, gqlSub } from "./client";
import { AbortableComponent } from "./refresh-component";

export interface Face {
  id: string;
  nid: number;
  locator: FaceLocator;
}

const counters = [
  "rxFrames", "rxInterests", "rxData", "rxNacks", "rxDecodeErrs", "rxReassPackets", "rxReassDrops",
  "txFrames", "txInterests", "txData", "txNacks", "txAllocErrs", "txFragGood", "txFragBad",
] as const;

interface Counters extends Record<typeof counters[number], number> {
}

interface Props {
  face: Face;
  rxQueues?: number[];
}

interface State {
  cnt: Counters;
}

export class FaceGrid extends AbortableComponent<Props, State> {
  state: State = {
    cnt: Object.fromEntries(counters.map((k) => [k, 0])) as any,
  };

  override async componentDidMount() {
    const { id } = this.props.face;
    for await (const { faceCounters } of gqlSub<{ faceCounters: Counters }>(gql`
      subscription threadLoadStat($id: ID!) {
        faceCounters(id: $id, interval: "1s", diff: true) {
          ${counters.join(" ")}
        }
      }
    `, { id }, this.abort)) {
      this.setState({ cnt: faceCounters });
    }
  }

  override render() {
    const { face: { id, nid, locator }, rxQueues } = this.props;
    const { rxFrames, rxInterests, rxData, rxNacks, rxDecodeErrs, rxReassPackets, rxReassDrops,
      txFrames, txInterests, txData, txNacks, txAllocErrs, txFragGood, txFragBad } = this.state.cnt;
    return (
      <svg style="margin: 5px; background: #ffffff; width: 220px; border: solid 1px #2ecc40;" viewBox="0 0 220 160">
        <text x="1" y="10">
          {nid}
          <title>{id}</title>
        </text>
        {rxQueues && <text x="219" y="10" text-anchor="end">{rxQueues.length === 1 ? "RX-queue" : "RX-queues"} {rxQueues.join(",")}</text>}
        <text x="1" y="20">
          {describeFaceLocator(locator)}
          <title>{JSON.stringify(locator, undefined, 2)}</title>
        </text>
        {this.renderV(0, 25, "f", "frames", rxFrames, "decode-err", rxDecodeErrs)}
        {this.renderV(0, 50, "I", "Interests", rxInterests)}
        {this.renderV(0, 75, "D", "Data", rxData)}
        {this.renderV(0, 100, "N", "Nacks", rxNacks)}
        {this.renderV(0, 125, "R", "reass", rxReassPackets, "reassembler dropped", rxReassDrops)}
        {this.renderV(120, 25, "f", "frames", txFrames, "alloc-err", txAllocErrs)}
        {this.renderV(120, 50, "I", "Interests", txInterests)}
        {this.renderV(120, 75, "D", "Data", txData)}
        {this.renderV(120, 100, "N", "Nacks", txNacks)}
        {this.renderV(120, 125, "F", "frag", txFragGood, "fragmenter dropped", txFragBad)}
      </svg>
    );
  }

  private renderV(x: number, y: number, short: string, long: string, v: number, warnDesc?: string, warnV?: number) {
    return (
      <g transform={`translate(${x} ${y})`}>
        <rect
          width="100" height="20"
          stroke="#aaaaaa" stroke-width="1" fill="#ffffff"
        />
        <text x="5" y="15">{short}<title>{long}</title></text>
        <text x="75" y="15" text-anchor="end">{v}</text>
        {warnDesc ? (
          <circle cx="90" cy="10" r="6" fill={warnV === 0 ? "#2ecc40" : "#ff4136"}>
            <title>{warnV} {warnDesc}</title>
          </circle>
        ) : undefined}
      </g>
    );
  }
}

function describeFaceLocator(loc: FaceLocator): string {
  switch (loc.scheme) {
    case "ether":
      return `Ethernet ${loc.remote}`;
    case "udpe":
      return `UDP ${loc.remoteIP}:${loc.remoteUDP}`;
    case "vxlan":
      return `VXLAN ${loc.remoteIP} ${loc.vxlan}`;
    case "unix":
    case "udp":
    case "tcp":
      return `${loc.scheme.toUpperCase()} socket ${loc.remote}`;
    case "memif":
      return `memif ${loc.socketName} ${loc.id}`;
    default:
      return JSON.stringify(loc);
  }
}
