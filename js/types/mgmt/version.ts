export interface VersionMgmt {
  Version: { args: {}; reply: VersionReply };
}

export interface VersionReply {
  Commit: string;
}
