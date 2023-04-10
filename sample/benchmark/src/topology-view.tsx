import { Component, h } from "preact";

import type { BenchmarkOptions, ServerEnv } from "./benchmark";

interface Props {
  env: ServerEnv;
  opts: BenchmarkOptions;
}

export class TopologyView extends Component<Props> {
  override render() {
    const {
      env: {
        A_GQLSERVER,
        B_GQLSERVER,
        F_PORT_A,
        F_PORT_B,
        A_PORT_F,
        B_PORT_F,
      },
      opts: {
        faceAScheme,
        faceARxQueues,
        faceBScheme,
        faceBRxQueues,
        nFwds,
        trafficDir,
        producerKind,
        nProducerThreads,
      },
    } = this.props;
    const oneGen = A_GQLSERVER === B_GQLSERVER;
    const pThreads = `${nProducerThreads}x ${producerKind}`;
    const cThreads = "fetchers";
    return (
      <svg width="500" height="200" style="background: #ffffff;">
        <g transform="translate(0 0)">
          <rect width="100" height="200" fill="#ffdc00"/>
          <text x="10" y="90">forwarder</text>
          <text x="10" y="110" font-size="80%">{nFwds}x fwds</text>
        </g>
        <rect hidden={!oneGen} x="300" y="0" width="200" height="200" fill="#ffdc00"/>
        <rect hidden={oneGen} x="300" y="0" width="200" height="95" fill="#ffdc00"/>
        <rect hidden={oneGen} x="300" y="105" width="200" height="95" fill="#ffdc00"/>
        <g transform="translate(300 0)">
          <text x="20" y="40">traffic gen A</text>
          <text x="20" y="60" font-size="80%">{`${(trafficDir === 2 ? [pThreads, cThreads] : [pThreads]).join(" + ")}`}</text>
        </g>
        <g transform="translate(300 105)">
          <text x="20" y="40">traffic gen B</text>
          <text x="20" y="60" font-size="80%">{`${(trafficDir === 2 ? [pThreads, cThreads] : [cThreads]).join(" + ")}`}</text>
        </g>
        {this.renderLink("face A", "/A", faceAScheme, faceARxQueues, F_PORT_A, A_PORT_F, 50)}
        {this.renderLink("face B", "/B", faceBScheme, faceBRxQueues, F_PORT_B, B_PORT_F, 150)}
      </svg>
    );
  }

  private renderLink(
      title: string,
      prefix: string,
      scheme: BenchmarkOptions.FaceScheme,
      nRxQueues: number,
      portL: string,
      portR: string,
      y: number,
  ) {
    return (
      <g transform={`translate(100 ${y})`}>
        <text x="0" y="0" text-anchor="start">{scheme === "memif" ? "memif" : portL}</text>
        <text x="200" y="0" text-anchor="end">{scheme === "memif" ? "memif" : portR}</text>
        <text x="200" y="20" text-anchor="end">{prefix}</text>
        <text x="100" y="20" text-anchor="middle">{title}</text>
        <line x1="0" y1="5" x2="200" y2="5" stroke="#001f3f" stroke-width="2"/>
        {Array.from({ length: nRxQueues }).map((x, i) => (
          <g key={i} transform={`translate(0 ${-5 * i})`}>
            <polygon points="200,6 210,3 200,0" fill="#2ecc40"/>
            <polygon points="0,6 -10,3 0,0" fill="#2ecc40"/>
            <title>RX thread</title>
          </g>
        ))}
        <g>
          <polygon points="210,12 200,9 210,6" fill="#2ecc40"/>
          <polygon points="-10,12 0,9 -10,6" fill="#2ecc40"/>
          <title>TX thread</title>
        </g>
      </g>
    );
  }
}
