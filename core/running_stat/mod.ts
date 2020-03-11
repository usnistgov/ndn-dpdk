export interface Snapshot {
  /**
   * Number of inputs.
   * @TJS-type integer
   * @minimum 0
   */
  Count: number;

  /**
   * Number of samples.
   * @TJS-type integer
   * @minimum 0
   */
  Len: number;

  /**
   * Minimum value.
   */
  Min?: number;

  /**
   * Maximum value.
   */
  Max?: number;

  /**
   * Mean.
   */
  Mean?: number;

  /**
   * Variance of samples.
   * @minimum 0
   */
  Variance?: number;

  /**
   * Standard deviation of samples.
   * @minimum 0
   */
  Stdev?: number;

  /**
   * Internal variable M1.
   */
  M1: number;

  /**
   * Internal variable M2.
   */
  M2: number;
}

function updateDerivedFields(v: Snapshot) {
  if (v.Len > 0) {
    v.Mean = v.M1;
  }
  if (v.Len > 1) {
    v.Variance = v.M2 / (v.Len - 1);
    v.Stdev = Math.sqrt(v.Variance);
  }
}

export const empty: Snapshot = {
  Count: 0,
  Len: 0,
  M1: 0,
  M2: 0,
};

/** Combine stats instances. */
export function add(a: Snapshot, b: Snapshot): Snapshot {
  if (a.Len === 0) {
    return b;
  } else if (b.Len === 0) {
    return a;
  }
  const cLen = a.Len + b.Len;
  const delta = b.M1 - a.M1;
  const delta2 = delta * delta;
  const c: Snapshot = {
    Count: a.Count + b.Count,
    Len: cLen,
    M1: (a.Len * a.M1 + b.Len * b.M1) / cLen,
    M2: a.M2 + b.M2 + delta2 * a.Len * b.Len / cLen,
  };
  updateDerivedFields(c);
  return c;
}

/** Subtract stats instances. */
export function sub(c: Snapshot, a: Snapshot): Snapshot {
  const bLen = c.Len - a.Len;
  const bM1 = (c.Len * c.M1 - a.Len * a.M1) / bLen;
  const delta = a.M1 - bM1;
  const delta2 = delta * delta;
  const b: Snapshot = {
    Count: c.Count - a.Count,
    Len: bLen,
    M1: bM1,
    M2: c.M2 - a.M2 - delta2 * a.Len * bLen / c.Len,
  };
  updateDerivedFields(b);
	return b;
}
