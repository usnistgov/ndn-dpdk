import EventEmitter = require("events");
import ndnjs = require("ndn-js");
import { ElementReader } from "ndn-js/js/encoding/element-reader.js";
import net = require("net");

const wf = ndnjs.TlvWireFormat.get();

export class AppConn extends EventEmitter {
  constructor(socket: net.Socket) {
    super();
    const er = new ElementReader(this);
    socket.on("data", er.onReceivedData.bind(er));
    socket.on("close", this.emit.bind(this, "close"));
  }

  private onReceivedElement(buf: ndnjs.Buffer) {
    switch (buf[0]) {
      case 0x05:
        const interest = new ndnjs.Interest();
        try {
          wf.decodeInterest(interest, buf);
          this.emit("ndninterest", interest, buf);
        } catch (ex) {}
        break;
      case 0x06:
        const data = new ndnjs.Data();
        try {
          wf.decodeData(data, buf);
          this.emit("ndndata", data, buf);
        } catch (ex) {}
        break;
    }
  }
}
