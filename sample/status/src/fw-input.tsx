import numd from "numd";
import { Fragment, h } from "preact";

import { client, gql } from "./client";
import { TimerRefreshComponent } from "./refresh-component";

interface Props {
  id: string;
}

interface RxGroup {
  __typename: string;
  faces: Array<{
    id: string;
  }>;
  port?: {
    id: string;
    nid: number;
    numaSocket?: number;
    name: string;
  };
  queue?: number;
}

interface State {
  worker?: {
    numaSocket: number;
  };
  rxGroups?: RxGroup[];
}

export class FwInput extends TimerRefreshComponent<Props, State> {
  state: State = {};

  protected override async refresh(): Promise<Partial<State> | undefined> {
    const node = await client.request<State>(gql`
      query FwInput($id: ID!) {
        node(id: $id) {
          ... on FwInput {
            worker { numaSocket }
            rxGroups {
              __typename
              faces { id nid }
              ... on EthRxgFlow {
                port { id nid numaSocket name }
                queue
              }
              ... on EthRxgTable {
                port { id nid numaSocket name }
                queue
              }
            }
          }
        }
      }
    `, { id: this.props.id }, { key: "node" });
    return node;
  }

  override render() {
    const { worker, rxGroups } = this.state;
    if (!worker || !rxGroups) {
      return undefined;
    }
    return (
      <>
        {rxGroups.map((rxg, i) => {
          const [color, title] = describeRxGroup(rxg);
          return (
            <rect key={i} x={1 + i * 20} y={30} width="15" height="15" fill={color}>
              <title>{title}</title>
            </rect>
          );
        })}
      </>
    );
  }
}

// https://learnui.design/tools/data-color-picker.html#divergent
const RxGroupColors: Record<string, string> = {
  EthRxgFlow: "#221f72",
  EthRxgTable: "#418ec2",
  _: "#cff8ff",
  SocketRxEpoll: "#76c1aa",
  SocketRxConns: "#5b843c",
};

function describeRxGroup(rxg: RxGroup): [color: string, title: string] {
  const color = RxGroupColors[rxg.__typename] ?? RxGroupColors._;
  let title = rxg.__typename;
  if (rxg.port) {
    title += ` on ${rxg.port?.name} queue ${rxg.queue}`;
  }
  title += `, ${numd(rxg.faces.length, "face", "faces")}`;
  return [color, title];
}
