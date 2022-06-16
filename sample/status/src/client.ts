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

export class GqlErrors extends Error {
  constructor(errs: ReadonlyArray<{}>) {
    super(errs.map((e) => e.toString()).join("\n"));
  }
}

export function gqlQuery<T extends {}>(query: string, variables?: Record<string, unknown>): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    let result!: T;
    client.subscribe({
      query,
      variables,
    }, {
      next({ data, errors }) {
        if (errors) {
          reject(new GqlErrors(errors));
        } else {
          result = data as T;
        }
      },
      error: reject,
      complete() { resolve(result); },
    });
  });
}

export async function* gqlSubError<T extends {}>(query: string, variables?: Record<string, unknown>, { signal }: { signal?: AbortSignal } = {}): AsyncIterable<T | GqlErrors> {
  const q = pushable<T | GqlErrors>({ objectMode: true });
  const unsubscribe = client.subscribe({
    query,
    variables,
  }, {
    next({ data, errors }) {
      if (errors) {
        q.push(new GqlErrors(errors));
      } else {
        q.push(data as T);
      }
    },
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

export async function* gqlSub<T extends {}>(query: string, variables?: Record<string, unknown>, { signal }: { signal?: AbortSignal } = {}): AsyncIterable<T> {
  for await (const item of gqlSubError<T>(query, variables, { signal })) {
    if (item instanceof GqlErrors) {
      throw item;
    } else {
      yield item;
    }
  }
}
