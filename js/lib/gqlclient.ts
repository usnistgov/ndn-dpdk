import { type ClientError, type RequestDocument, gql, GraphQLWebSocketClient } from "graphql-request";
import WebSocket from "isomorphic-ws";
import { pushable } from "it-pushable";
import throat from "throat";

export { gql };

/** NDN-DPDK GraphQL client. */
export class GqlClient {
  /**
   * Constructor.
   * @param uri NDN-DPDK GraphQL server URI.
   */
  constructor(uri: string | URL) {
    uri = new URL(uri, globalThis.document?.URL);
    uri.protocol = uri.protocol.replace(/^http/, "ws");
    this.uri = uri.toString();
  }

  private readonly mutex = throat(1);
  private uri: string;
  public client?: GraphQLWebSocketClient;

  private async reconnect(): Promise<void> {
    this.client ??= await this.mutex(async () => new Promise<GraphQLWebSocketClient>((resolve, reject) => {
      const ws = new WebSocket(this.uri, GraphQLWebSocketClient.PROTOCOL);
      ws.addEventListener("error", (evt) => reject(evt.error));
      ws.addEventListener("close", () => { this.client = undefined; });

      const client = new GraphQLWebSocketClient(ws as any, {
        async onAcknowledged() { resolve(client); },
      });
    }));
  }

  /** Close the GraphQL client. */
  public close(): void {
    this.client?.close();
    this.client = undefined;
  }

  /** Run a query or mutation. */
  public async request<T>(query: RequestDocument, vars: Record<string, any> = {}, {
    signal,
    key,
  }: GqlClient.Options = {}): Promise<T> {
    await this.reconnect();
    let value = await this.client!.request(query, vars);
    if (signal?.aborted) {
      throw signal.reason as Error;
    }
    if (key) {
      value = value[key];
    }
    return value;
  }

  /** Run the delete mutation. */
  public del(id: string) {
    return this.request<boolean>(gql`
      mutation delete($id: ID!) {
        delete(id: $id)
      }
    `, { id }, {
      key: "delete",
    });
  }

  /** Run a subscription. */
  public async *subscribe<T>(query: RequestDocument, vars: Record<string, any> = {}, {
    signal,
    key,
    onError,
  }: GqlClient.SubscribeOptions = {}): AsyncIterable<T> {
    await this.reconnect();
    const q = pushable<T>({ objectMode: true });
    const unsubscribe = this.client!.subscribe(query, {
      next: (value) => {
        if (key) {
          value = value[key];
        }
        q.push(value);
      },
      error: (err) => {
        if (onError) {
          onError(err);
        } else {
          q.end(err);
        }
      },
      complete: () => {
        q.end();
      },
    }, vars);

    signal?.addEventListener("abort", unsubscribe);
    try {
      yield* q;
    } finally {
      unsubscribe();
    }
  }
}

export namespace GqlClient {
  export interface Options {
    /** AbortSignal to cancel the GraphQL operation. */
    signal?: AbortSignal;

    /** If specified, extract a top-level field from the result value. */
    key?: string;
  }

  export interface SubscribeOptions extends Options {
    /** If specified, receive GraphQL error via callback instead of canceling the subscription. */
    onError?: (err: ClientError) => void;
  }
}
