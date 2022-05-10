import { Component, h } from "preact";

import type { WorkerRole } from "./model";

const roleColor: Record<WorkerRole, string> = {
  FWD: "#0074d9",
  RX: "#3d9970",
  TX: "#3d9970",
  CRYPTO: "#ffdc00",
  DISK: "#ffdc00",
  PRODUCER: "#0074d9",
  CONSUMER: "#0074d9",
};

type makeRolePath = (w: number, h: number) => string;

function makeRolePathRect(w: number, h: number) {
  return `M 0 0 V ${h} H ${w} V 0 H 0`;
}

const rolePath: Partial<Record<WorkerRole, makeRolePath>> = {
  RX: (w, h) => `M 0 0 V ${h} L ${w} ${h * 0.9} V ${h * 0.1} L 0 0`,
  TX: (w, h) => `M ${w} 0 V ${h} L 0 ${h * 0.9} V ${h * 0.1} L ${w} 0`,
};

interface Props {
  role: WorkerRole;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
}

export class WorkerShape extends Component<Props> {
  override render() {
    const { role, label, x, y, width, height, children } = this.props;
    return (
      <g transform={`translate(${x} ${y})`}>
        <path
          d={(rolePath[role] ?? makeRolePathRect)(width, height)}
          stroke={roleColor[role]} stroke-width="1" fill="transparent"
        />
        <text x="1" y={height * 0.1} dominant-baseline="hanging">{label}</text>
        {children}
      </g>
    );
  }
}
