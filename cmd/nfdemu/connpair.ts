import net = require("net");

import { AppConn } from "./appconn";
import { FwConn } from "./fwconn";

export class ConnPair {
  private ac: AppConn;
  private fc: FwConn;

  constructor(appSocket: net.Socket) {
    this.fc = new FwConn();
    this.fc.on("connected", () => {
      console.log(">CONNECTED");
    });

    this.ac = new AppConn(appSocket);
    this.ac.on("ndninterest", (interest, buf) => {
      console.log("<I ", interest.getName().toUri());
      this.fc.send(buf);
    });
    this.ac.on("ndndata", (data, buf) => {
      console.log("<D ", data.getName().toUri());
      // TODO lookup and insert PIT token
    });
    this.ac.on("close", () => {
      console.log("<CLOSE");
      this.fc.close();
    });
  }
}
