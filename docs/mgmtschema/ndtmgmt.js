(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareType('ndt.Value', {
  type: 'integer',
  minimum: 0,
  maximum: 255,
});

ctx.declareMethod('Ndt.ReadTable', true, 'blob');

ctx.declareMethod('Ndt.ReadCounters', true,
  {
    type: 'array',
    items: ctx.useType('counter'),
  });

ctx.declareMethod('Ndt.Update',
  {
    type: 'object',
    properties: {
      Value: ctx.useType('ndt.Value'),
    },
    required: ['Value'],
    anyOf: [
      {
        properties: {
          Hash: ctx.useType('counter'),
        },
        required: ['Hash'],
      },
      {
        properties: {
          Name: ctx.useType('ndn.Name'),
        },
        required: ['Name'],
      },
    ],
  },
  {
    type: 'object',
    properties: {
      Index: ctx.useType('counter'),
    }
  });

};
})(exports);
