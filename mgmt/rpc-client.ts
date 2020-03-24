import * as jayson from "jayson";
import { URL } from "url";

import { Mgmt } from "./schema";

/** Wrapper of jayson.Client that provides async API. */
export class RpcClient {
  constructor(private readonly jaysonClient: jayson.Client) {
  }

  public async request<M extends keyof Mgmt, V extends keyof Mgmt[M],
    A extends Mgmt[M][V] extends { args: infer A } ? A : never,
    R extends Mgmt[M][V] extends {reply: infer R}?R:never,
  >(module: M, method: V, args: A): Promise<R> {
    return new Promise<R>((resolve, reject) => {
      this.jaysonClient.request(`${module}.${method}`, args as jayson.RequestParamsLike,
        (err, error, result: R) => {
          const e = err ?? error;
          if (e) {
            reject(e);
            return;
          }
          resolve(result);
        });
    });
  }
}

export function makeMgmtClient(mgmtUri?: string): RpcClient {
  const mgmtEnv = mgmtUri ?? process.env.MGMT ?? "tcp://127.0.0.1:6345";
  if (mgmtEnv === "0") {
    throw new Error("management socket disabled");
  }

  const u = new URL(mgmtEnv);
  if (!/^tcp[46]?:$/.test(u.protocol)) {
    throw new Error(`unsupported MGMT scheme ${u.protocol}`);
  }

  const jaysonClient = jayson.Client.tcp({
    host: u.hostname,
    port: parseInt(u.port, 10),
  });
  return new RpcClient(jaysonClient);
}
