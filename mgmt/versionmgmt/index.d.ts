export as namespace versionmgmt;

export interface VersionReply {
  Commit: string;
}

export interface VersionMgmt {
  Version: {args: {}, reply: VersionReply};
}
