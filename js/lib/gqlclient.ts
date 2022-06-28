import { type ClientError, type RequestDocument, gql, GraphQLWebSocketClient } from "graphql-request";
import WebSocket from "isomorphic-ws";
import { pushable } from "it-pushable";
import pDefer from "p-defer";

export { gql };

/** NDN-DPDK GraphQL client. */
export class GqlClient {
  /**
   * Constructor.
   * @param uri NDN-DPDK GraphQL server URI.
   */
  constructor(uri: string | URL) {
    const connecting = pDefer<void>();
    this.connecting = connecting.promise;

    uri = new URL(uri);
    uri.protocol = uri.protocol.replace(/^http/, "ws");
    const ws = new WebSocket(uri.toString(), GraphQLWebSocketClient.PROTOCOL);
    ws.addEventListener("error", (evt) => connecting.reject(evt.error));
    ws.addEventListener("close", (evt) => connecting.reject(new Error("WebSocket closed")));

    this.client = new GraphQLWebSocketClient(ws as any, {
      async onAcknowledged() { connecting.resolve(); },
    });
  }

  private readonly connecting: Promise<void>;
  public readonly client: GraphQLWebSocketClient;

  /** Close the GraphQL client. */
  public close(): void {
    this.client.close();
  }

  /** Run a query or mutation. */
  public async request<T>(query: RequestDocument, vars: Record<string, any> = {}, {
    signal,
    key,
  }: GqlClient.Options = {}): Promise<T> {
    await this.connecting;
    let value = await this.client.request(query, vars);
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
    await this.connecting;
    const q = pushable<T>({ objectMode: true });
    const unsubscribe = this.client.subscribe(query, {
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
