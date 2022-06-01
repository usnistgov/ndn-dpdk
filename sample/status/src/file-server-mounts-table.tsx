import type { FileServerMount } from "@usnistgov/ndn-dpdk";
import { Component, h } from "preact";

import { formatName } from "./model";

interface Props {
  mounts: FileServerMount[];
}

export class FileServerMountsTable extends Component<Props> {
  override render() {
    const { mounts } = this.props;
    return (
      <table class="pure-table pure-table-horizontal">
        <thead>
          <tr>
            <th>prefix</th>
            <th>path</th>
          </tr>
        </thead>
        <tbody>
          {mounts.map(({ prefix, path }) => (
            <tr key={prefix}>
              <td>{formatName(prefix)}</td>
              <td>{path}</td>
            </tr>
          ))}
        </tbody>
      </table>
    );
  }
}
