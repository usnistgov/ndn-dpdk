(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareType('fwdpmgmt.IndexArg', ctx.markAllRequired({
  type: 'object',
  properties: {
    Index: ctx.useType('integer'),
  },
}));

ctx.declareMethod('DPInfo.Global', 'null',
  {
    type: 'object',
    properties: {
      NInputs: ctx.useType('counter'),
      NFwds: ctx.useType('counter'),
    },
  });

ctx.declareMethod('DPInfo.Input', 'fwdpmgmt.IndexArg',
  {
    type: 'object',
    properties: {
      LCore: ctx.useType('dpdk.LCore'),
      Faces: ctx.useType('iface.FaceId[]'),
      NNameDisp: ctx.useType('counter'),
      NTokenDisp: ctx.useType('counter'),
      NBadToken: ctx.useType('counter'),
    },
  });

ctx.declareMethod('DPInfo.Fwd', 'fwdpmgmt.IndexArg',
  {
    type: 'object',
    properties: {
      LCore: ctx.useType('dpdk.LCore'),
      QueueCapacity: ctx.useType('counter'),
      NQueueDrops: ctx.useType('counter'),
      InputLatency: ctx.useType('running_stat.Snapshot'),
      HeaderMpUsage: ctx.useType('counter'),
      IndirectMpUsage: ctx.useType('counter'),
    },
  });

ctx.declareMethod('DPInfo.Pit', 'fwdpmgmt.IndexArg',
  {
    type: 'object',
    properties: {
      NEntries: ctx.useType('counter'),
      NInsert: ctx.useType('counter'),
      NFound: ctx.useType('counter'),
      NCsMatch: ctx.useType('counter'),
      NAllocErr: ctx.useType('counter'),
      NDataHit: ctx.useType('counter'),
      NDataMiss: ctx.useType('counter'),
      NNackHit: ctx.useType('counter'),
      NNackMiss: ctx.useType('counter'),
      NExpired: ctx.useType('counter'),
    },
  });

ctx.declareMethod('DPInfo.Cs', 'fwdpmgmt.IndexArg',
  {
    type: 'object',
    properties: {
      Capacity: ctx.useType('counter'),
      NEntries: ctx.useType('counter'),
      NHits: ctx.useType('counter'),
      NMisses: ctx.useType('counter'),
    },
  });

};
})(exports);
