import type { FaceLocator } from "@usnistgov/ndn-dpdk";
import { Component, Fragment, h } from "preact";

import { gql, gqlQuery } from "./client";
import { FaceGrid } from "./face-grid";

interface EthDev {
  id: string;
  nid: number;
  name: string;
  numaSocket: number;
  macAddr: string;
  mtu: number;
  rxImpl?: string;
  isDown: boolean;
}

interface Face {
  ethDev?: Pick<EthDev, "id">;
  id: string;
  nid: number;
  locator: FaceLocator;
}

interface State {
  ethDevs: EthDev[];
  faces: Face[];
}

export class FacesList extends Component<{}, State> {
  state: State = {
    ethDevs: [],
    faces: [],
  };

  override async componentDidMount() {
    const result = await gqlQuery<State>(gql`
      {
        ethDevs { id nid name numaSocket macAddr mtu rxImpl isDown }
        faces { ethDev { id } id nid locator }
      }
    `);
    this.setState(result);
  }

  override render() {
    return (
      <>
        {this.state.ethDevs.map((ethDev) => this.renderEthDev(ethDev))}
        {this.renderEthDev(undefined)}
      </>
    );
  }

  private renderEthDev(ethDev?: EthDev) {
    const faces = this.state.faces.filter((face) => face.ethDev?.id === ethDev?.id);
    if (!ethDev && faces.length === 0) {
      return undefined;
    }
    return (
      <section>
        <h3>{ethDev ? `${ethDev.name} (${ethDev.macAddr} ${ethDev.rxImpl})` : "Non-Ethernet faces"}</h3>
        <div style="display: flex; flex-direction: row; flex-wrap: wrap;">
          {faces.map((face) => <FaceGrid key={face.id} face={face}/>)}
        </div>
      </section>
    );
  }
}
