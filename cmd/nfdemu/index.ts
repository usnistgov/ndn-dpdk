import fs = require("fs");
import net = require("net");

import { ConnPair } from "./connpair";

const listenerPath = "/tmp/nfdemu.sock";
if (fs.existsSync(listenerPath)) {
  fs.unlinkSync(listenerPath);
}

const server = new net.Server();
server.on("connection", (socket: net.Socket) => { new ConnPair(socket); });
server.listen(listenerPath);
