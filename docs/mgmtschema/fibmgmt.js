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
    {
      properties: {
        HasEntry: { const:false },
      },
    },
    {
      required: ['Name', 'Nexthops'],
    },
  ]
});

ctx.declareMethod('Fib.Info', 'null',
  {
    type: 'object',
    properties: {
      NEntries: ctx.useType('counter'),
      NEntriesDup: ctx.useType('counter'),
      NVirtuals: ctx.useType('counter'),
      NNodes: ctx.useType('counter'),
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

};
})(exports);
