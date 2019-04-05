import * as fs from "fs";
import * as jayson from "jayson";
import * as _ from "lodash";
import ndn = require("ndn-js");
import * as net from "net";

import * as mgmt from "../../docs/mgmttypes";
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
      fs.unlink(this.path, _.noop);
      this.accept(socket);
    });
    this.server.listen(this.path);
    this.faceId = 0;

    mgmtClient.request("Face.Create",
      [
        {
          RemoteUri: "unix://" + this.path,
        },
      ] as mgmt.facemgmt.CreateArg,
      (err, error, result: mgmt.facemgmt.CreateRes) => {
        if (err || error || result.length < 1) {
          return;
        }
        this.faceId = result[0].Id;
        this.emit("faceidready", this.faceId);
      });
  }

  public registerPrefix(name: ndn.Name, cb: (ok: boolean) => void): void {
    if (!this.faceId) {
      this.once("faceidready", () => { this.registerPrefix(name, cb); });
      return;
    }
    const cb2 = _.once(cb);
    mgmtClient.request("Fib.Insert",
      {
        Name: name.toUri(),
        Nexthops: [this.faceId],
      } as mgmt.fibmgmt.InsertArg,
      (err, error, result: mgmt.fibmgmt.InsertRes) => {
        const ok = !(err || error);
        cb2(ok);
      });
  }

  public close(): boolean {
    if (!super.close()) {
      return false;
    }
    mgmtClient.request("Face.Destroy",
      {
        Id: this.faceId,
      } as mgmt.facemgmt.DestroyArg,
      _.noop);
    return true;
  }
}
