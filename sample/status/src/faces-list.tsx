import type { FaceLocator } from "@usnistgov/ndn-dpdk";
import { Fragment, h } from "preact";

import { gql, gqlQuery } from "./client";
import { FaceGrid } from "./face-grid";
import { TimerRefreshComponent } from "./refresh-component";

interface EthDev {
  id: string;
  nid: number;
  name: string;
  numaSocket: number;
  macAddr: string;
  mtu: number;
  rxImpl?: string;
  isDown: boolean;
  devInfo: {
    driver?: string;
  };
  rxGroups: Array<{
    faces: Array<Pick<Face, "id">>;
    queue?: number;
  }>;
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

export class FacesList extends TimerRefreshComponent<{}, State> {
  state: State = {
    ethDevs: [],
    faces: [],
  };

  protected override refresh() {
    return gqlQuery<State>(gql`
      {
        ethDevs {
          id nid name numaSocket macAddr mtu rxImpl isDown devInfo
          rxGroups { queue faces { id } }
        }
        faces { ethDev { id } id nid locator }
      }
    `);
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
      <section key={ethDev?.id ?? ""}>
        <h3>{ethDev ? `${ethDev.name} (${ethDev.macAddr} ${ethDev.devInfo.driver} ${ethDev.rxImpl})` : "Non-Ethernet faces"}</h3>
        <div style="display: flex; flex-direction: row; flex-wrap: wrap;">
          {faces.map((face) => {
            const rxQueues = ethDev?.rxGroups.filter(
              (rxg) => rxg.faces.some(({ id }) => id === face.id) && rxg.queue !== undefined,
            ).map(({ queue }) => queue!);
            return (
              <FaceGrid key={face.id} face={face} rxQueues={rxQueues?.length ? rxQueues : undefined}/>
            );
          })}
        </div>
      </section>
    );
  }
}
