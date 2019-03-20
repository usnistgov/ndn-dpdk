import * as fs from "fs";
import * as net from "net";

import { Transfer } from "./transfer";

const listenerPath = "/tmp/nfdemu.sock";
if (fs.existsSync(listenerPath)) {
  fs.unlinkSync(listenerPath);
}

const server = new net.Server();
server.on("connection", (socket: net.Socket) => { new Transfer(socket).begin(); });
server.listen(listenerPath);
