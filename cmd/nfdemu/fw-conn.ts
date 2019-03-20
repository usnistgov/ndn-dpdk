import * as fs from "fs";
import * as jayson from "jayson";
import ndn = require("ndn-js");
import * as net from "net";
import { noop } from "node-noop";

import { SocketConn } from "./socket-conn";

const mgmtClient = jayson.Client.tcp({port: 6345});

export class FwConn extends SocketConn {
  public id: number;
  private path: string;
  private server: net.Server;
  private faceId: number;

  constructor() {
    super();

    this.id = Math.floor(Math.random() * 100000000);
    this.path = "/tmp/nfdemu-" + this.id + ".sock";
    this.server = new net.Server();
    this.server.once("connection", (socket: net.Socket) => {
      this.server.close();
      fs.unlink(this.path, noop);
      this.accept(socket);
    });
    this.server.listen(this.path);
    this.faceId = 0;

    mgmtClient.request("Face.Create",
      {
        RemoteUri: "unix://" + this.path,
      },
      (err, response) => {
        if (response && response.result) {
          this.faceId = response.result.Id;
          this.emit("faceidready", this.faceId);
        }
      });
  }

  public registerPrefix(name: ndn.Name, cb: (bool) => void): void {
    if (!this.faceId) {
      this.once("faceidready", () => { this.registerPrefix(name, cb); });
      return;
    }
    let done = false;
    mgmtClient.request("Fib.Insert",
      {
        Name: name.toUri(),
        Nexthops: [this.faceId],
      },
      (err, response) => {
        if (!done) {
          cb(!!(response && response.result));
        }
        done = true;
      });
  }

  public close(): boolean {
    if (!super.close()) {
      return false;
    }
    mgmtClient.request("Face.Destroy",
      {
        Id: this.faceId,
      },
      noop);
    return true;
  }
}
