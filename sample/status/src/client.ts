import { gql } from "graphql-request";
import { createClient } from "graphql-ws";
import { pushable } from "it-pushable";

export { gql };

const url = new URL("/graphql", document.URL);
url.protocol = url.protocol.replace(/^http/, "ws");
export const client = createClient({
  url: url.toString(),
  lazy: false,
});

export function gqlQuery<T extends {}>(query: string, variables?: Record<string, unknown>): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let result!: T;
    client.subscribe({
      query,
      variables,
    }, {
      next({ data }) { result = data as T; },
      error: reject,
      complete() { resolve(result); },
    });
  });
}

export async function* gqlSub<T extends {}>(query: string, variables?: Record<string, unknown>, { signal }: { signal?: AbortSignal } = {}): AsyncIterable<T> {
  const q = pushable<T>();
  const unsubscribe = client.subscribe({
    query,
    variables,
  }, {
    next({ data }) { q.push(data as T); },
    error(err) { q.end(err as Error); },
    complete() { q.end(); },
  });
  signal?.addEventListener("abort", unsubscribe);
  try {
    yield* q;
  } finally {
    unsubscribe();
  }
}
