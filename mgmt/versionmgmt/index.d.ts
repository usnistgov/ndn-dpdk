export as namespace versionmgmt;

export interface VersionReply {
  Commit: string;
  BuildTime: Date;
}

export interface VersionMgmt {
  Version: {args: {}, reply: VersionReply};
}
