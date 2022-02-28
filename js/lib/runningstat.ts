import type { RunningStatSnapshot as SnapshotJSON } from "../types/core";

function nullNaN(v: number): number | undefined {
  return Number.isNaN(v) ? undefined : v;
}

const empty: Readonly<SnapshotJSON> = {
  count: 0,
  len: 0,
  m1: 0,
  m2: 0,
};

export class Snapshot {
  constructor(v: SnapshotJSON = empty) {
    this.count = BigInt(v.count);
    this.len = BigInt(v.len);
    this.min = v.min;
    this.max = v.min;
    this.m1 = v.m1;
    this.m2 = v.m2;
  }

  count: bigint;
  len: bigint;
  min?: number;
  max?: number;
  m1: number;
  m2: number;

  public get mean(): number {
    return this.len > 0n ? this.m1 : Number.NaN;
  }

  public get variance(): number {
    return this.len > 1n ? this.m2 / Number(this.len - 1n) : Number.NaN;
  }

  public get stdev(): number {
    return Math.sqrt(this.variance);
  }

  public toJSON(): SnapshotJSON {
    const j: SnapshotJSON = {
      count: this.count.toString(),
      len: this.len.toString(),
      min: this.min,
      max: this.max,
      mean: nullNaN(this.mean),
      variance: nullNaN(this.variance),
      stdev: nullNaN(this.stdev),
      m1: this.m1,
      m2: this.m2,
    };
    return j;
  }

  /** Combine stats instances. */
  public add(other: Readonly<Snapshot>): Snapshot {
    if (this.len === 0n) {
      return new Snapshot(other.toJSON());
    }
    if (other.len === 0n) {
      return new Snapshot(this.toJSON());
    }

    const sum = new Snapshot();
    sum.count = this.count + other.count;
    sum.len = this.len + other.len;

    const delta = other.m1 - this.m1;
    const delta2 = delta * delta;
    sum.m1 = (Number(this.len) * this.m1 + Number(other.len) * other.m1) / Number(sum.len);
    sum.m2 = this.m2 + other.m2 + delta2 * Number(this.len * other.len) / Number(sum.len);
    return sum;
  }

  /** Subtract stats instances. */
  public sub(this: Readonly<Snapshot>, other: Readonly<Snapshot>): Snapshot {
    const diff = new Snapshot();
    diff.count = this.count - other.count;
    diff.len = this.len - other.len;

    const diffM1 = (Number(this.len) * this.m1 - Number(other.len) * other.m1) / Number(diff.len);
    const delta = other.m1 - diffM1;
    const delta2 = delta * delta;
    diff.m1 = diffM1;
    diff.m2 = this.m2 - other.m2 - delta2 * Number(other.len * diff.len) / Number(this.len);
    return diff;
  }
}
