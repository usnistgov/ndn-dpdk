import * as net from "net";

import { SocketConn } from "./socket-conn";

export class AppConn extends SocketConn {
  public begin: () => void;

  constructor(socket: net.Socket) {
    super();
    this.begin = () => { this.accept(socket); };
  }
}
