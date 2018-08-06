import net = require("net");

import { AppConn, FwConn } from "./conn";
import { PktType } from "./packet";

export class ConnPair {
  private ac: AppConn;
  private fc: FwConn;

  constructor(appSocket: net.Socket) {
    this.fc = new FwConn();
    this.fc.on("connected", () => {
      console.log(">CONNECTED");
    });
    this.fc.on("packet", (pkt) => {
      console.log(">" + pkt.toString());
      this.ac.send(pkt.buf);
    });

    this.ac = new AppConn(appSocket);
    this.ac.on("packet", (pkt) => {
      console.log("<" + pkt.toString());
      this.fc.send(pkt.buf);
    });
    this.ac.on("close", () => {
      console.log("<CLOSE");
      this.fc.close();
    });
  }
}
