import delay = require("delay");
import * as fs from "fs";
import * as jayson from "jayson";
import * as _ from "lodash";
import ndn = require("ndn-js");
import * as net from "net";

import * as iface from "../../iface";
import * as facemgmt from "../../mgmt/facemgmt";
import * as fibmgmt from "../../mgmt/fibmgmt";

import { SocketConn } from "./socket-conn";

const mgmtClient = jayson.Client.tcp({port: 6345});

export class FwConn extends SocketConn {
  public faceId: number;

  constructor() {
    super();
    this.faceId = 0;
    const path = "/tmp/nfdemu-" + Math.floor(Math.random() * 100000000) + ".sock";
    Promise.all([
      this.listen(path),
      delay(10).then(() => this.faceCreate(path)),
    ])
    .then(([socket, faceId]) => {
      this.faceId = faceId;
      this.accept(socket);
    })
    .catch((reason) => {
      this.emit("error", reason);
    });
  }

  public async registerPrefix(name: ndn.Name): Promise<void> {
    if (!this.faceId) {
      throw new Error("not connected");
    }
    return new Promise<void>((resolve, reject) => {
      mgmtClient.request("Fib.Insert",
        {
          Name: name.toUri(),
          Nexthops: [this.faceId],
        } as fibmgmt.InsertArg,
        (err, error, result: fibmgmt.InsertReply) => {
          if (err || error) {
            reject(err || error);
          } else {
            resolve();
          }
        });
    });
  }

  public close(): boolean {
    if (!super.close()) {
      return false;
    }
    mgmtClient.request("Face.Destroy",
      {
        Id: this.faceId,
      } as facemgmt.IdArg,
      _.noop);
    return true;
  }

  private listen(path: string): Promise<net.Socket> {
    return new Promise<net.Socket>((resolve, reject) => {
      const server = new net.Server();
      server.once("connection", (socket: net.Socket) => {
        server.close();
        fs.unlink(path, _.noop);
        resolve(socket);
      });
      server.listen({ path, exclusive: true });
    });
  }

  private faceCreate(path: string): Promise<number> {
    return new Promise<number>((resolve, reject) => {
      mgmtClient.request("Face.Create",
        {
          Remote: path,
          Scheme: "unix",
        } as iface.Locator,
        (err, error, result: facemgmt.BasicInfo) => {
          if (err || error) {
            reject(err || error);
            return;
          }
          resolve(result.Id);
        });
    });
  }
}
