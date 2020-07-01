import { TcpTransportClient } from "mole-rpc-transport-tcp";
// @ts-expect-error
import MoleClient = require("mole-rpc/MoleClient");
import { URL } from "url";

import type { Mgmt } from "../types/mgmt/mod";

// https://github.com/microsoft/TypeScript/issues/36295 workaround
function makeDefaultMgmtUri() {
  return process.env.MGMT ?? "tcp://127.0.0.1:6345";
}

/** Management RPC client. */
export class RpcClient {
  public static create(mgmtUri = makeDefaultMgmtUri()): RpcClient {
    if (mgmtUri === "0") {
      throw new Error("management socket disabled");
    }
    const { protocol, hostname, port } = new URL(mgmtUri);
    if (!/^tcp[46]?:$/.test(protocol)) {
      throw new Error(`unsupported MGMT scheme ${protocol}`);
    }

    const transport = new TcpTransportClient({
      host: hostname,
      port: Number.parseInt(port, 10),
    });
    const client = new MoleClient({
      requestTimeout: 3600000,
      transport,
    });
    return new RpcClient(transport, client);
  }

  private constructor(private readonly transport: TcpTransportClient, private readonly client: any) {
  }

  public async request<M extends keyof Mgmt, V extends keyof Mgmt[M],
    A extends Mgmt[M][V] extends { args: infer A } ? A : never,
    R extends Mgmt[M][V] extends { reply: infer R } ? R : never,
  >(module: M, method: V, arg: A): Promise<R> {
    const params = { ...(arg as object) };
    Object.defineProperty(params, "length", { value: true });
    return this.client.callMethod(`${module}.${method}`, params);
  }

  public close() {
    this.transport.close();
  }
}
