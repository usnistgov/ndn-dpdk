(function(exports){
exports.provideDefinitions = function(declareType, useType, declareMethod) {

declareType('versionmgmt.VersionReply', {
  type: 'object',
  properties: {
    Commit: {
      type: 'string',
      pattern: '/^[0-9a-f]{40}$/',
    },
    BuildTime: {
      type: 'string',
      format: 'date-time',
    },
  },
});
declareMethod('Version.Version', 'null', 'versionmgmt.VersionReply');

};
})(exports);
