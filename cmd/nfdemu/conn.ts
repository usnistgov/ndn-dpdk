import EventEmitter = require("events");
import fs = require("fs");
import jayson = require("jayson");
import { ElementReader as ndnjs_ElementReader } from "ndn-js/js/encoding/element-reader.js";
import net = require("net");

import { Packet } from "./packet";

const mgmtClient = jayson.client.tcp({port: 6345});

class SocketConn extends EventEmitter {
  public get isConnected(): boolean { return !!this.socket; }
  protected socket: net.Socket;
  private er: ndnjs_ElementReader;

  public send(buf: Buffer): void {
    if (!this.isConnected) {
      return;
    }
    this.socket.write(buf);
  }

  public close(): boolean {
    if (!this.isConnected) {
      return false;
    }

    this.socket.end();
    this.socket = null;
    this.emit("close");
    return true;
  }

  protected accept(socket: net.Socket) {
    this.socket = socket;
    this.er = new ndnjs_ElementReader(this);
    this.socket.on("data", (buf: Buffer) => { this.er.onReceivedData(buf); });
    this.socket.on("close", () => {
      if (!this.isConnected) {
        return;
      }
      this.socket = null;
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
  private path: string;
  private server: net.Server;
  private faceId: number;

  constructor() {
    super();

    this.path = "/tmp/nfdemu-" + Math.floor(Math.random() * 100000000) + ".sock";
    this.server = new net.Server();
    this.server.once("connection", (socket: net.Socket) => {
      this.server.close();
      fs.unlink(this.path, () => {});
      this.accept(socket);
    });
    this.server.listen(this.path);

    mgmtClient.request("Face.Create",
      {
        RemoteUri: "unix://" + this.path,
      },
      (err, response) => {
        if (response && response.result) {
          this.faceId = response.result.Id;
        }
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
      () => {});
    return true;
  }
}
