import * as jayson from "jayson";

/**
 * Wrapper of jayson.Client that provides async API.
 */
export class RpcClient {
  private jaysonClient: jayson.Client;

  constructor(jaysonClient: jayson.Client) {
    this.jaysonClient = jaysonClient;
  }

  public async request<A extends jayson.RequestParamsLike,
                       R extends jayson.JSONRPCResultLike>(method: string, args: A): Promise<R> {
    return new Promise<R>((resolve, reject) => {
      this.jaysonClient.request(method, args,
        (err, error, result: R) => {
          if (err || error) {
            reject(err || error);
            return;
          }
          resolve(result);
        });
    });
  }
}
