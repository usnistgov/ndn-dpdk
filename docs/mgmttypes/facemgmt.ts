export namespace facemgmt {
  export interface localRemoteUris {
    LocalUri: string;
    RemoteUri: string;
  }

  export interface IdArg {
    Id: number;
  }

  export type BasicInfo = IdArg & localRemoteUris;

  export type CreateArg = localRemoteUris[];
  export type CreateRes = ReadonlyArray<BasicInfo>;

  export type DestroyArg = IdArg;
}
