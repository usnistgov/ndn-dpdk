import ndn = require("ndn-js");
import * as net from "net";

import { AppConn } from "./app-conn";
import { FwConn } from "./fw-conn";
import { Packet, PktType } from "./packet";
import { PendingInterest } from "./pending-interest";
import { PendingInterestList } from "./pending-interest-list";

const ndnjs = ndn as any;

const keyChain = new ndn.KeyChain("pib-memory:", "tpm-memory:");
const signingInfo = new ndnjs.SigningInfo(ndnjs.SigningInfo.SignerType.SHA256);
const ribRegisterPrefix = new ndn.Name("/localhost/nfd/rib/register");

export class Transfer {
  public get id(): number { return this.fc.id; }
  private fc: FwConn;
  private ac: AppConn;
  private pil: PendingInterestList;

  constructor(appSocket: net.Socket) {
    this.pil = new PendingInterestList();
    this.fc = new FwConn();
    this.ac = new AppConn(appSocket);
  }

  public begin(): void {
    this.fc.on("connected", () => {
      this.log(">", "CONNECTED");
    });
    this.fc.on("packet", (pkt: Packet) => { this.handleFwPacket(pkt); });

    this.ac.on("close", () => {
      this.log("<", "CLOSE");
      this.fc.close();
    });
    this.ac.on("packet", (pkt: Packet) => { this.handleAppPacket(pkt); });
  }

  private log(direction: string, name: string, pitToken: string = ""): void {
    // tslint:disable-next-line no-console
    console.log("%d %s%s %s", this.id, direction, name, pitToken);
  }

  private handleFwPacket(pkt: Packet): void {
    this.log(">", pkt.toString(), pkt.pitToken);
    if (pkt.type === PktType.Interest) {
      this.pil.insert(new PendingInterest(pkt.interest!, pkt.pitToken));
    }
    this.ac.send(pkt.wireEncode(false));
  }

  private handleAppPacket(pkt: Packet): void {
    switch (pkt.type) {
      case PktType.Interest:
        if (ribRegisterPrefix.match(pkt.interest!.getName())) {
          this.handleAppPrefixReg(pkt.interest!);
          return;
        }
        break;
      case PktType.Data:
      case PktType.Nack:
        const pi = this.pil.find(pkt.name!);
        if (pi) {
          pkt.pitToken = pi.pitToken;
        }
        break;
    }
    this.log("<", pkt.toString(), pkt.pitToken);
    this.fc.send(pkt.wireEncode(true));
  }

  private handleAppPrefixReg(interest: ndn.Interest): void {
    const cp = new ndnjs.ControlParameters();
    cp.wireDecode(interest.getName().get(ribRegisterPrefix.size()));
    this.log("<R ", cp.getName().toUri());
    this.fc.registerPrefix(cp.getName(), (ok: boolean) => {
      const cr = new ndnjs.ControlResponse();
      if (ok) {
        this.log(">R ", cp.getName().toUri());
        const flags = new ndnjs.ForwardingFlags();
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
