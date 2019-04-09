import ndn = require("ndn-js");

import { PendingInterest } from "./pending-interest";

export class PendingInterestList {
  public get length() { return this.list.length; }
  private list: PendingInterest[];

  constructor() {
    this.list = [];
  }

  public insert(pi: PendingInterest): void {
    this.cleanup();
    this.list.push(pi);
  }

  public find(name: ndn.Name): PendingInterest|undefined {
    this.cleanup();
    for (let i = 0; i < this.length; ++i) {
      const pi = this.list[i];
      if (pi.name.match(name)) {
        this.list.splice(i, 1);
        return pi;
      }
    }
    return undefined;
  }

  private cleanup(): void {
    const now = new Date();
    while (this.length && this.list[0].expiry < now) {
      this.list.shift();
    }
  }
}
