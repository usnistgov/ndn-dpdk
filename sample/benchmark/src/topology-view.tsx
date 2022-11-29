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
      A_GQLSERVER,
      B_GQLSERVER,
      F_PORT_A,
      F_PORT_B,
      A_PORT_F,
      B_PORT_F,
      F_CORES_PRIMARY: { length: fCores },
      A_CORES_PRIMARY: { length: aCores },
      B_CORES_PRIMARY: { length: bCores },
    } = this.props.env;
    const oneGen = A_GQLSERVER === B_GQLSERVER;
    return (
      <svg width="400" height="200" style="background: #ffffff;">
        <g transform="translate(0 0)">
          <rect width="100" height="200" fill="#ffdc00"/>
          <text x="50" y="90" text-anchor="middle">forwarder</text>
          <text x="50" y="120" text-anchor="middle">({fCores} cores)</text>
        </g>
        <g transform="translate(300 0)" hidden={!oneGen}>
          <rect width="100" height="200" fill="#ffdc00"/>
          <text x="50" y="90" text-anchor="middle">traffic gen</text>
          <text x="50" y="120" text-anchor="middle">({aCores} cores)</text>
        </g>
        <g transform="translate(300 0)" hidden={oneGen}>
          <rect width="100" height="95" fill="#ffdc00"/>
          <text x="50" y="40" text-anchor="middle">traffic gen A</text>
          <text x="50" y="70" text-anchor="middle">({aCores} cores)</text>
        </g>
        <g transform="translate(300 105)" hidden={oneGen}>
          <rect width="100" height="95" fill="#ffdc00"/>
          <text x="50" y="40" text-anchor="middle">traffic gen B</text>
          <text x="50" y="70" text-anchor="middle">({bCores} cores)</text>
        </g>
        <g transform="translate(100 50)">
          <text x="0" y="0" text-anchor="start">{stripVlan(F_PORT_A)}</text>
          <text x="200" y="0" text-anchor="end">{stripVlan(A_PORT_F)}</text>
          <text x="200" y="20" text-anchor="end">/A</text>
          <text x="100" y="20" text-anchor="middle">face A</text>
          <line x1="0" y1="5" x2="200" y2="5" stroke="#001f3f" stroke-width="2"/>
        </g>
        <g transform="translate(100 150)">
          <text x="0" y="0" text-anchor="start">{stripVlan(F_PORT_B)}</text>
          <text x="200" y="0" text-anchor="end">{stripVlan(B_PORT_F)}</text>
          <text x="200" y="20" text-anchor="end">/B</text>
          <text x="100" y="20" text-anchor="middle">face B</text>
          <line x1="0" y1="5" x2="200" y2="5" stroke="#001f3f" stroke-width="2"/>
        </g>
      </svg>
    );
  }
}
