(function(exports){
exports.provideDefinitions = function(declareType, useType, declareMethod) {

declareType('fibmgmt.NameArg', {
  type: 'object',
  properties: {
    Name: useType('ndn.Name'),
  },
});

declareType('fibmgmt.LookupReply', {
  type: 'object',
  properties: {
    HasEntry: useType('boolean'),
    Name: useType('ndn.Name'),
    Nexthops: useType('array-of_iface.FaceId'),
  },
});

declareMethod('Fib.Info', 'null',
  {
    type: 'object',
    properties: {
      NEntries: useType('counter'),
      NVirtuals: useType('counter'),
    },
  });

declareMethod('Fib.List', 'null',
  {
    type: 'array',
    items: useType('ndn.Name'),
  });

declareMethod('Fib.Insert',
  {
    type: 'object',
    properties: {
      Name: useType('ndn.Name'),
      Nexthops: useType('array-of_iface.FaceId'),
    },
  },
  {
    type: 'object',
    properties: {
      IsNew: useType('boolean'),
    },
  });

declareMethod('Fib.Erase', 'fibmgmt.NameArg', 'null');

declareMethod('Fib.Find', 'fibmgmt.NameArg', 'fibmgmt.LookupReply');

declareMethod('Fib.Lpm', 'fibmgmt.NameArg', 'fibmgmt.LookupReply');

};
})(exports);
