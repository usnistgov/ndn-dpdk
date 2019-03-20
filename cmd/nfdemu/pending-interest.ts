import ndn = require("ndn-js");

export class PendingInterest {
  public name: ndn.Name;
  public expiry: Date;
  public pitToken?: string;

  constructor(interest: ndn.Interest, pitToken?: string) {
    this.name = interest.getName();
    const lifetime = interest.getInterestLifetimeMilliseconds() || 4000;
    this.expiry = new Date(new Date().getTime() + lifetime);
    this.pitToken = pitToken;
  }
}
