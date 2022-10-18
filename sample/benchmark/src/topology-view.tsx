import { Component, h } from "preact";

import type { ServerEnv } from "./benchmark";

interface Props {
  env: ServerEnv;
}

function stripVlan(s: string): string {
  return s.split("+")[0];
}

export class TopologyView extends Component<Props> {
  override render() {
    const {
      F_PORTS: [fPortA, fPortB],
      F_CORES_PRIMARY: { length: fCores },
      G_PORTS: [gPortA, gPortB],
      G_CORES_PRIMARY: { length: gCores },
    } = this.props.env;
    return (
      <svg width="400" height="200" style="background: #ffffff;">
        <g transform="translate(0 0)">
          <rect width="100" height="200" fill="#ffdc00"/>
          <text x="50" y="90" text-anchor="middle">forwarder</text>
          <text x="50" y="120" text-anchor="middle">({fCores} cores)</text>
        </g>
        <g transform="translate(300 0)">
          <rect width="100" height="200" fill="#ffdc00"/>
          <text x="50" y="90" text-anchor="middle">traffic gen</text>
          <text x="50" y="120" text-anchor="middle">({gCores} cores)</text>
        </g>
        <g transform="translate(100 50)">
          <text x="0" y="0" text-anchor="start">{stripVlan(fPortA)}</text>
          <text x="200" y="0" text-anchor="end">{stripVlan(gPortA)}</text>
          <text x="200" y="20" text-anchor="end">/A</text>
          <line x1="0" y1="5" x2="200" y2="5" stroke="#001f3f" stroke-width="2"/>
        </g>
        <g transform="translate(100 150)">
          <text x="0" y="0" text-anchor="start">{stripVlan(fPortB)}</text>
          <text x="200" y="0" text-anchor="end">{stripVlan(gPortB)}</text>
          <text x="200" y="20" text-anchor="end">/B</text>
          <line x1="0" y1="5" x2="200" y2="5" stroke="#001f3f" stroke-width="2"/>
        </g>
      </svg>
    );
  }
}
