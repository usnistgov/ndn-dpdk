import EventEmitter = require("events");
import jayson = require("jayson");
import ndnjs = require("ndn-js");
import net = require("net");

const mgmtClient = jayson.client.tcp({port: 6345});

export class FwConn extends EventEmitter {
  public isConnected: boolean;
  private path: string;
  private server: net.Server;
  private socket: net.Socket;
  private faceId: number;

  constructor() {
    super();

    this.isConnected = false;
    this.path = "/tmp/nfdemu-" + Math.floor(Math.random() * 100000000) + ".sock";
    this.server = new net.Server();
    this.server.on("connection", (socket: net.Socket) => {
      this.socket = socket;
      this.emit("connected");
      this.isConnected = true;
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

  public send(buf): void {
    if (!this.isConnected) {
      return;
    }
    this.socket.write(buf);
  }

  public close(): void {
    if (!this.isConnected) {
      return;
    }
    this.isConnected = false;
    mgmtClient.request("Face.Destroy",
      {
        Id: this.faceId,
      },
      (err, response) => {});
  }
}
