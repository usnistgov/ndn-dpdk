import EventEmitter = require("events");
import { ElementReader as ndn_ElementReader } from "ndn-js/js/encoding/element-reader.js";
import * as net from "net";
import { noop } from "node-noop";

import { Packet } from "./packet";

export class SocketConn extends EventEmitter {
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
