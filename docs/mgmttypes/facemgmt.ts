export namespace facemgmt {
  export interface ethfaceLocator {
    Scheme: "ether";
    Port: string;
    Local: string;
    Remote: string;
  }

  export interface socketfaceLocator {
    Scheme: "udp"|"unixgram"|"tcp"|"unix";
    Local: string;
    Remote: string;
  }

  export type Locator = ethfaceLocator|socketfaceLocator;

  export interface IdArg {
    Id: number;
  }

  export interface BasicInfo extends IdArg {
    Locator: Locator;
  }

  export type CreateArg = Locator;
  export type CreateRes = BasicInfo;

  export type DestroyArg = IdArg;
}
