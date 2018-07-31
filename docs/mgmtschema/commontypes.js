(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareType('counter', {
  type: 'integer',
  minimum: 0,
})

ctx.declareType('blob', {
  type: 'string',
  contentEncoding: 'base64',
  contentMediaType: 'application/octet-stream',
})

ctx.declareType('running_stat.Snapshot', {
  type: 'object',
  properties: {
    Count: ctx.useType('counter'),
    Min: ctx.useType('number'),
    Max: ctx.useType('number'),
    Mean: ctx.useType('number'),
    Stdev: {
      type: 'number',
      minimum: 0,
    },
  }
});

ctx.declareType('dpdk.LCore', {
  type: 'integer',
  minimum: 0,
  maximum: 127,
});

ctx.declareType('iface.FaceId', {
  type: 'integer',
  minimum: 1,
  maximum: 65535,
});

ctx.declareType('iface.FaceId[]', {
  type: 'array',
  items: ctx.useType('iface.FaceId'),
  uniqueItems: true,
});

ctx.declareType('strategycode.Id', {
  type: 'integer',
  minimum: 1,
})

ctx.declareType('ndn.Name', {
  type: 'string',
  format: 'uri-reference',
});

};
})(exports);
