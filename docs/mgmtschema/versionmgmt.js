(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareMethod('Version.Version', true,
  {
    type: 'object',
    properties: {
      Commit: {
        type: 'string',
        pattern: '^[0-9a-f]{40}$',
      },
      BuildTime: {
        type: 'string',
        format: 'date-time',
      },
    },
    required: ['Commit'],
  });

};
})(exports);
