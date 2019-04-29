export namespace facemgmt {
  export interface localRemoteUris {
    LocalUri: string;
    RemoteUri: string;
  }

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

  export type CreateArg = localRemoteUris[];
  export type CreateRes = ReadonlyArray<BasicInfo>;

  export type DestroyArg = IdArg;
}
