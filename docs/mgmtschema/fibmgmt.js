(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareType('fibmgmt.NameArg', ctx.markAllRequired({
  type: 'object',
  properties: {
    Name: ctx.useType('ndn.Name'),
  },
}));

ctx.declareType('fibmgmt.Nexthops', {
  type: 'array',
  items: ctx.useType('iface.FaceId'),
  minItems: 1,
  uniqueItems: true,
});

ctx.declareType('fibmgmt.LookupReply', {
  type: 'object',
  properties: {
    HasEntry: ctx.useType('boolean'),
    Name: ctx.useType('ndn.Name'),
    Nexthops: ctx.useType('fibmgmt.Nexthops'),
  },
  anyOf: [
    ctx.markAllRequired({
      properties: {
        HasEntry: { const:false },
      },
    }),
    {
      required: ['Name', 'Nexthops'],
    },
  ],
});

ctx.declareType('fib.EntryCounters', {
  type: 'object',
  properties: {
    HasEntry: ctx.useType('boolean'),
    Name: ctx.useType('ndn.Name'),
    Nexthops: ctx.useType('fibmgmt.Nexthops'),
  },
});

ctx.declareMethod('Fib.Info', 'null',
  {
    type: 'object',
    properties: {
      NRxInterests: ctx.useType('counter'),
      NRxData: ctx.useType('counter'),
      NRxNacks: ctx.useType('counter'),
      NTxInterests: ctx.useType('counter'),
    },
  });

ctx.declareMethod('Fib.List', 'null',
  {
    type: 'array',
    items: ctx.useType('ndn.Name'),
  });

ctx.declareMethod('Fib.Insert',
  ctx.markAllRequired({
    type: 'object',
    properties: {
      Name: ctx.useType('ndn.Name'),
      Nexthops: ctx.useType('fibmgmt.Nexthops'),
    },
  }),
  {
    type: 'object',
    properties: {
      IsNew: ctx.useType('boolean'),
    },
  });

ctx.declareMethod('Fib.Erase', 'fibmgmt.NameArg', 'null');

ctx.declareMethod('Fib.Find', 'fibmgmt.NameArg', 'fibmgmt.LookupReply');

ctx.declareMethod('Fib.Lpm', 'fibmgmt.NameArg', 'fibmgmt.LookupReply');

ctx.declareMethod('Fib.ReadEntryCounters', 'fibmgmt.NameArg', 'fib.EntryCounters');

};
})(exports);
