import assert from "minimalistic-assert";

export function uniqueRandomVector(count: number, max: number): number[] {
  assert(count <= max);
  const vec: number[] = [];
  for (let i = 0; i < max; ++i) {
    vec.push(i);
  }
  const order = globalThis.crypto.getRandomValues(new Uint32Array(max));
  vec.sort((a, b) => order[a] - order[b]);
  return vec.slice(0, count);
}

export function hexPad(n: number | bigint, len: number): string {
  return n.toString(16).toUpperCase().padStart(len, "0");
}
