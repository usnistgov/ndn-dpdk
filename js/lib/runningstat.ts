import type { RunningStatSnapshot } from "../types/core";

export type Snapshot = RunningStatSnapshot;

function updateDerivedFields(v: Snapshot) {
  if (v.len > 0) {
    v.mean = v.m1;
  }
  if (v.len > 1) {
    v.variance = v.m2 / (v.len - 1);
    v.stdev = Math.sqrt(v.variance);
  }
}

export const empty: Readonly<Snapshot> = {
  count: 0,
  len: 0,
  m1: 0,
  m2: 0,
};

/** Combine stats instances. */
export function add(a: Readonly<Snapshot>, b: Readonly<Snapshot>): Snapshot {
  if (a.len === 0) {
    return { ...b };
  } if (b.len === 0) {
    return { ...a };
  }
  const cLen = a.len + b.len;
  const delta = b.m1 - a.m1;
  const delta2 = delta * delta;
  const c: Snapshot = {
    count: a.count + b.count,
    len: cLen,
    m1: (a.len * a.m1 + b.len * b.m1) / cLen,
    m2: a.m2 + b.m2 + delta2 * a.len * b.len / cLen,
  };
  updateDerivedFields(c);
  return c;
}

/** Subtract stats instances. */
export function sub(c: Readonly<Snapshot>, a: Readonly<Snapshot>): Snapshot {
  const bLen = c.len - a.len;
  const bM1 = (c.len * c.m1 - a.len * a.m1) / bLen;
  const delta = a.m1 - bM1;
  const delta2 = delta * delta;
  const b: Snapshot = {
    count: c.count - a.count,
    len: bLen,
    m1: bM1,
    m2: c.m2 - a.m2 - delta2 * a.len * bLen / c.len,
  };
  updateDerivedFields(b);
  return b;
}
