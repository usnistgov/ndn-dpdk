import EventEmitter = require("events");
import * as fs from "fs";
import * as jayson from "jayson";
import ndn = require("ndn-js");
import { ElementReader as ndn_ElementReader } from "ndn-js/js/encoding/element-reader.js";
import * as net from "net";
import { noop } from "node-noop";

import { Packet } from "./packet";

const mgmtClient = jayson.client.tcp({port: 6345});

class SocketConn extends EventEmitter {
  public get isConnected(): boolean { return !!this.socket; }
  protected socket?: net.Socket;
  private er: ndn_ElementReader;

  public send(buf: Buffer): void {
    if (!this.isConnected) {
      this.once("connected", () => { this.send(buf); });
      return;
    }
    this.socket!.write(buf);
  }

  public close(): boolean {
    if (!this.isConnected) {
      return false;
    }

    this.socket!.end();
    this.socket = undefined;
    this.emit("close");
    return true;
  }

  protected accept(socket: net.Socket) {
    this.socket = socket;
    this.er = new ndn_ElementReader(this);
    this.socket.on("data", (buf: Buffer) => { this.er.onReceivedData(buf); });
    this.socket.on("error", noop);
    this.socket.on("close", () => {
      if (!this.isConnected) {
        return;
      }
      this.socket = undefined;
      this.emit("close");
    });

    this.emit("connected");
  }

  protected onReceivedElement(buf: Buffer) {
    this.emit("packet", new Packet(buf));
  }
}

export class AppConn extends SocketConn {
  constructor(socket: net.Socket) {
    super();
    this.accept(socket);
  }
}

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
