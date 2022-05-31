// @ts-expect-error no typing
import { get as hashGet } from "hashquery";
import { Component, Fragment, h, render } from "preact";

import { gql, gqlQuery } from "./client";
import { FacesList } from "./faces-list";
import { FwDiagram } from "./fw-diagram";
import { Worker, WorkerRole, WorkersByRole } from "./model";
import { TgDiagram } from "./tg-diagram";
import { WorkersTable } from "./workers-table";

const roles = {
  "": "(unactivated)",
  fw: "forwarder",
  tg: "traffic generator",
};

const tabs = {
  diagram: ({ role }: State) => {
    switch (role) {
      case "fw":
        return <FwDiagram/>;
      case "tg":
        return <TgDiagram/>;
    }
    return undefined;
  },
  faces: () => <FacesList/>,
  workers: ({ workers }: State) => <WorkersTable workers={workers}/>,
};

interface State {
  role: keyof typeof roles;
  tab: keyof typeof tabs;
  workers: WorkersByRole;
}

class App extends Component<{}, State> {
  state: State = {
    role: "",
    tab: "diagram",
    workers: {},
  };

  override async componentDidMount() {
    this.handleHashChange();
    window.addEventListener("hashchange", this.handleHashChange);

    const { workers } = await gqlQuery<{ workers: ReadonlyArray<Worker<WorkerRole | "">> }>(gql`
      {
        workers { ${Worker.subselection} }
      }
    `);
    const sw: State["workers"] = {};
    for (const w of workers) {
      if (!w.role) {
        continue;
      }
      (sw[w.role] ??= []).push(w as Worker);
    }
    const role = sw.FWD ? "fw" : (sw.PRODUCER || sw.CONSUMER) ? "tg" : "";
    document.title = `NDN-DPDK ${roles[role]} status`;
    this.setState({
      role,
      workers: sw,
    });
  }

  override componentWillUnmount() {
    window.removeEventListener("hashchange", this.handleHashChange);
  }

  override render() {
    return (
      <>
        <nav class="pure-menu pure-menu-horizontal">
          <span class="pure-menu-heading">NDN-DPDK</span>
          <ul class="pure-menu-list">
            {Object.keys(tabs).map((tab) => (
              <li key={tab} class={`pure-menu-item ${tab === this.state.tab ? "pure-menu-selected" : ""}`}>
                <a href={`#tab=${tab}`} class="pure-menu-link">{tab}</a>
              </li>
            ))}
          </ul>
        </nav>
        {tabs[this.state.tab]?.(this.state)}
      </>
    );
  }

  private readonly handleHashChange = () => {
    const tab = hashGet("tab");
    if (tab in tabs) {
      this.setState({ tab });
    }
  };
}

document.addEventListener("DOMContentLoaded", () => render(<App/>, document.body));
