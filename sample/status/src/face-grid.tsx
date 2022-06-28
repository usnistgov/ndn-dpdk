import numd from "numd";
import { Fragment, h } from "preact";

import { client, gql } from "./client";
import { type Face, describeFaceLocator } from "./model";
import { AbortableComponent } from "./refresh-component";

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
    for await (const faceCounters of client.subscribe<Counters>(gql`
      subscription faceCounters($id: ID!) {
        faceCounters(id: $id, interval: "1s", diff: true) {
          ${counters.join(" ")}
        }
      }
    `, { id }, { signal: this.signal, key: "faceCounters" })) {
      this.setState({ cnt: faceCounters });
    }
  }

  override render() {
    const { face: { id, nid, locator, isDown }, rxQueues } = this.props;
    const { rxFrames, rxInterests, rxData, rxNacks, rxDecodeErrs, rxReassPackets, rxReassDrops,
      txFrames, txInterests, txData, txNacks, txAllocErrs, txFragGood, txFragBad } = this.state.cnt;
    return (
      <svg style="margin: 5px; background: #ffffff; width: 220px; border: solid 1px #2ecc40;" viewBox="0 0 220 160">
        <text x="1" y="10">
          {nid}
          <title>{id}</title>
        </text>
        <text x="1" y="20">
          {describeFaceLocator(locator)}
          <title>{JSON.stringify(locator, undefined, 2)}</title>
        </text>
        <text x="1" y="159">
          {
            rxQueues ? (
              <>
                {`RX (${numd(rxQueues.length, " queue", " queues")})`}
                <title>{rxQueues.join(", ")}</title>
              </>
            ) : "RX"
          }
        </text>
        {this.renderV(0, 25, "frame", rxFrames, "decode error", rxDecodeErrs)}
        {this.renderV(0, 50, "Interest", rxInterests)}
        {this.renderV(0, 75, "Data", rxData)}
        {this.renderV(0, 100, "Nack", rxNacks)}
        {this.renderV(0, 125, "reass", rxReassPackets, "reassembler dropped", rxReassDrops)}
        <text x="219" y="159" text-anchor="end">TX</text>
        {this.renderV(120, 25, "frame", txFrames, "alloc error", txAllocErrs)}
        {this.renderV(120, 50, "Interest", txInterests)}
        {this.renderV(120, 75, "Data", txData)}
        {this.renderV(120, 100, "Nack", txNacks)}
        {this.renderV(120, 125, "frag", txFragGood, "fragmenter dropped", txFragBad)}
        <path hidden={!isDown} fill="#ff4136" fill-opacity="100" transform="translate(210 0)" d="M -5 0 V 10 H -10 L 0 15 L 10 10 H 5 V 0 Z">
          <title>face is down</title>
        </path>
      </svg>
    );
  }

  private renderV(x: number, y: number, desc: string, v: number, warnDesc?: string, warnV?: number) {
    return (
      <g transform={`translate(${x} ${y})`}>
        <rect
          width="100" height="20"
          stroke="#aaaaaa" stroke-width="1" fill="#ffffff"
        />
        <text x="1" y="15">{desc}</text>
        <text x="79" y="15" text-anchor="end">{v}</text>
        {warnDesc ? (
          <circle cx="90" cy="10" r="6" fill={warnV === 0 ? "#2ecc40" : "#ff4136"}>
            <title>{warnV} {warnDesc}</title>
          </circle>
        ) : undefined}
      </g>
    );
  }
}
