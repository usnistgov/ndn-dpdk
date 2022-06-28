import { h } from "preact";

import { client, gql } from "./client";
import { TimerRefreshComponent } from "./refresh-component";
import { Face, TgFace } from "./tg-face";

interface State {
  faces?: Face[];
}

export class TgDiagram extends TimerRefreshComponent<{}, State> {
  protected override async refresh() {
    const faces = await client.request<Array<Partial<Face>>>(gql`
      {
        faces {
          ${Face.subselection}
        }
      }
    `, {}, { key: "faces" });
    return {
      faces: faces.filter((face): face is Face => !!face.trafficgen),
    };
  }

  override render() {
    if (!this.state.faces) {
      return undefined;
    }
    const { faces } = this.state;
    return (
      faces.map((face) => (
        <TgFace key={face.id} face={face}/>
      ))
    );
  }
}
