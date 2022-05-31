import { h } from "preact";

import { gql, gqlQuery } from "./client";
import { TimerRefreshComponent } from "./refresh-component";
import { Face, TgFace } from "./tg-face";

interface TgQueryResult {
  faces: Face[];
}

interface State {
  faces?: Face[];
}

export class TgDiagram extends TimerRefreshComponent<{}, State> {
  protected override async refresh() {
    let { faces } = await gqlQuery<TgQueryResult>(gql`
      {
        faces {
          ${Face.subselection}
        }
      }
    `);
    faces = faces.filter((face) => !!(face as Partial<Face>).trafficgen);
    return { faces };
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
