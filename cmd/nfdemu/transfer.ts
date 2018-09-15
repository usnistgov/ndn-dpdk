import ndn = require("ndn-js");
import * as net from "net";

import { AppConn, FwConn } from "./conn";
import { Packet, PktType } from "./packet";

class PendingInterest {
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

class PendingInterestList {
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
      if (pi.name.isPrefixOf(name)) {
        this.list.splice(i, 1);
      }
      return pi;
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

const keyChain = new ndn.KeyChain("pib-memory:", "tpm-memory:");
const signingInfo = new ndn.SigningInfo(ndn.SigningInfo.SignerType.SHA256);
const ribRegisterPrefix = new ndn.Name("/localhost/nfd/rib/register");

export class Transfer {
  public get id(): number { return this.fc.id; }
  private fc: FwConn;
  private ac: AppConn;
  private pil: PendingInterestList;

  constructor(appSocket: net.Socket) {
    this.pil = new PendingInterestList();

    this.fc = new FwConn();
    this.fc.on("connected", () => {
      this.log(">", "CONNECTED");
    });
    this.fc.on("packet", (pkt: Packet) => { this.handleFwPacket(pkt); });

    this.ac = new AppConn(appSocket);
    this.ac.on("close", () => {
      this.log("<", "CLOSE");
      this.fc.close();
    });
    this.ac.on("packet", (pkt: Packet) => { this.handleAppPacket(pkt); });
  }

  private log(direction: string, name: string, pitToken: string = ""): void {
    console.log("%d %s%s %s", this.id, direction, name, pitToken);
  }

  private handleFwPacket(pkt: Packet): void {
    this.log(">", pkt.toString(), pkt.pitToken);
    if (pkt.type === PktType.Interest) {
      this.pil.insert(new PendingInterest(pkt.interest, pkt.pitToken));
    }
    this.ac.send(pkt.wireEncode(false));
  }

  private handleAppPacket(pkt: Packet): void {
    switch (pkt.type) {
      case PktType.Interest:
        if (ribRegisterPrefix.isPrefixOf(pkt.interest.getName())) {
          this.handleAppPrefixReg(pkt.interest);
          return;
        }
        break;
      case PktType.Data:
      case PktType.Nack:
        const pi = this.pil.find(pkt.name);
        if (pi) {
          pkt.pitToken = pi.pitToken;
        }
        break;
    }
    this.log("<", pkt.toString(), pkt.pitToken);
    this.fc.send(pkt.wireEncode(true));
  }

  private handleAppPrefixReg(interest: ndn.Interest): void {
    const cp = new ndn.ControlParameters();
    cp.wireDecode(interest.getName().getComponent(ribRegisterPrefix.size()));
    this.log("<R ", cp.getName().toUri());
    this.fc.registerPrefix(cp.getName(), (ok: boolean) => {
      const cr = new ndn.ControlResponse();
      if (ok) {
        this.log(">R ", cp.getName().toUri());
        const flags = new ndn.ForwardingFlags();
        flags.setChildInherit(false);
        flags.setCapture(true);
        cr.setStatusCode(200).setStatusText("OK");
        cp.setFaceId(1);
        cp.setOrigin(0);
        cp.setCost(0);
        cp.setForwardingFlags(flags);
        cr.setBodyAsControlParameters(cp);
      } else {
        cr.setStatusCode(500).setStatusText("ERROR");
      }

      const data = new ndn.Data();
      data.setName(interest.getName());
      data.setContent(cr.wireEncode());
      keyChain.sign(data, signingInfo);
      this.ac.send(data.wireEncode().buf());
    });
  }
}
