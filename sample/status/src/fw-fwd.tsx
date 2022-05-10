import { Component, Fragment, h } from "preact";

import { gql, gqlSub } from "./client";

interface Props {
  id: string;
}

const fwdCounters = [
  "nInterestsQueued", "nInterestsDropped", "nInterestsCongMarked",
  "nDataQueued", "nDataDropped", "nDataCongMarked",
  "nNacksQueued", "nNacksDropped", "nNacksCongMarked",
] as const;
const pitCounters = [
  "nEntries", "nInsert", "nExpired", "nAllocErr", "nDataMiss",
] as const;
const csCounters = [
  "directEntries", "directCapacity", "indirectEntries", "indirectCapacity",
  "nHitMemory", "nHitDisk", "nHitIndirect", "nMiss",
] as const;

interface State {
  fwd: Record<typeof fwdCounters[number], number>;
  pit: Record<typeof pitCounters[number], number>;
  cs: Record<typeof csCounters[number], number>;
}

export class FwFwd extends Component<Props, State> {
  state: State = {
    fwd: Object.fromEntries(fwdCounters.map((k) => [k, 0])) as any,
    pit: Object.fromEntries(pitCounters.map((k) => [k, 0])) as any,
    cs: Object.fromEntries(csCounters.map((k) => [k, 0])) as any,
  };

  private readonly abort = new AbortController();

  override componentDidMount() {
    void this.subscribe("fwd", gql`
      subscription fwFwdCounters($id: ID!) {
        result: fwFwdCounters(id: $id, interval: "1s") {
          ${fwdCounters.join(" ")}
        }
      }
    `, fwdCounters);
    void this.subscribe("pit", gql`
      subscription fwPitCounters($id: ID!) {
        result: fwPitCounters(id: $id, interval: "1s") {
          ${pitCounters.join(" ")}
        }
      }
    `, pitCounters.slice(1));
    void this.subscribe("cs", gql`
      subscription fwCsCounters($id: ID!) {
        result: fwCsCounters(id: $id, interval: "1s") {
          ${csCounters.join(" ")}
        }
      }
    `, csCounters.slice(4));
  }

  private async subscribe<K extends keyof State>(key: K, query: string, sub: ReadonlyArray<keyof State[K]>) {
    const { id } = this.props;
    let prev: State[K] = JSON.parse(JSON.stringify(this.state[key]));
    for await (const { result } of gqlSub<{ result: typeof prev }>(query, { id }, this.abort)) {
      const update: typeof prev = JSON.parse(JSON.stringify(result));
      for (const k of sub) {
        (update[k] as any) -= (prev[k] as any);
      }
      prev = result;
      this.setState({ [key]: update });
    }
  }

  override componentWillUnmount() {
    this.abort.abort();
  }

  override render() {
    return (
      <>
        {this.renderQueue(25, "Interests")}
        {this.renderQueue(50, "Data")}
        {this.renderQueue(75, "Nacks")}
        {this.renderPit()}
        {this.renderCs()}
      </>
    );
  }

  private renderQueue(y: number, t: "Interests" | "Data" | "Nacks") {
    const queued = this.state.fwd[`n${t}Queued`];
    const dropped = this.state.fwd[`n${t}Dropped`];
    const congMarked = this.state.fwd[`n${t}CongMarked`];
    return (
      <g transform={`translate(-2 ${y})`}>
        <rect
          width="100" height="20"
          stroke="#ff851b" stroke-width="1" fill="#ffffff"
        />
        <text x="5" y="15">{t[0]}<title>{t}</title></text>
        <text x="75" y="15" text-anchor="end">{queued}</text>
        <circle cx="90" cy="10" r="6" fill={dropped + congMarked === 0 ? "#2ecc40" : "#ff4136"}>
          <title>{dropped} dropped, {congMarked} congestion-marked</title>
        </circle>
      </g>
    );
  }

  private renderPit() {
    const { nEntries, nAllocErr, nDataMiss } = this.state.pit;
    return (
      <g transform={"translate(120 10)"}>
        <rect width="80" height="30" stroke="#ff851b" stroke-width="1" fill="transparent"/>
        <text x="1" y="10">PIT</text>
        <text x="1" y="28">{nEntries} entries</text>
        <circle cx="70" cy="15" r="6" fill={nAllocErr + nDataMiss === 0 ? "#2ecc40" : "#ff4136"}>
          <title>{nAllocErr} alloc-error, {nDataMiss} unsolicited Data</title>
        </circle>
      </g>
    );
  }

  private renderCs() {
    const { directEntries, directCapacity, nHitMemory, nHitDisk, nHitIndirect, nMiss } = this.state.cs;
    const totalHit = nHitMemory + nHitDisk + nHitIndirect;
    const hitDivisor = Math.max(1, totalHit + nMiss);
    return (
      <g transform={"translate(250 50)"}>
        <text x="-40" y="-30">CS</text>
        <circle r="40" stroke="#ff851b" stroke-width="1" fill="#ffffff"/>
        {this.renderRing(39, [
          ["#2ecc40", nHitMemory / hitDivisor, `${nHitMemory} memory hits`],
          ["#ffdc00", nHitDisk / hitDivisor, `${nHitDisk} disk hits`],
          ["#aaaaaa", nHitIndirect / hitDivisor, `${nHitIndirect} indirect hits`],
        ], `${totalHit} hits, ${nMiss} misses`)}
        {this.renderRing(15, [
          ["#7fdbff", directEntries / Math.max(1, directCapacity)],
        ], `${directEntries} / ${directCapacity} occupied`)}
      </g>
    );
  }

  private renderRing(radius: number, wedges: Array<[color: string, value: number, title?: string]>, title: string) {
    const c = Math.PI * radius;
    let sumValue = 0;
    return (
      <g>
        <title>{title}</title>
        <circle r={radius} fill="#ffffff"/>
        {wedges.map(([color, value, title]) => {
          const $circle = (
            <circle
              r={radius / 2} stroke={color} stroke-width={radius} fill="transparent"
              stroke-dasharray={`${c * value} ${c * (1 - value)}`}
              transform={`rotate(${360 * sumValue - 90})`}
            >
              {title ? <title>{title}</title> : undefined}
            </circle>
          );
          sumValue += value;
          return $circle;
        })}
      </g>
    );
  }
}
