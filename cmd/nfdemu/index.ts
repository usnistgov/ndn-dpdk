import fs = require("fs");
import net = require("net");

import { Transfer } from "./transfer";

const listenerPath = "/tmp/nfdemu.sock";
if (fs.existsSync(listenerPath)) {
  fs.unlinkSync(listenerPath);
}

const server = new net.Server();
server.on("connection", (socket: net.Socket) => { new Transfer(socket); });
server.listen(listenerPath);
