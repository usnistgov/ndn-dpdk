import * as net from "net";

import { SocketConn } from "./socket-conn";

export class AppConn extends SocketConn {
  constructor(socket: net.Socket) {
    super();
    this.accept(socket);
  }
}
