(function(exports){
exports.provideDefinitions = function(declareType, useType, declareMethod) {

declareType('iface.FaceUri', {
  type: 'string',
  format: 'uri',
});

declareType('iface.Counters', {
  type: 'object',
  properties: {
    RxFrames: useType('counter'),
    RxOctets: useType('counter'),
    L2DecodeErrs: useType('counter'),
    ReassBad: useType('counter'),
    ReassGood: useType('counter'),
    L3DecodeErrs: useType('counter'),
    RxInterests: useType('counter'),
    RxData: useType('counter'),
    RxNacks: useType('counter'),
    FragGood: useType('counter'),
    FragBad: useType('counter'),
    TxAllocErrs: useType('counter'),
    TxQueued: useType('counter'),
    TxDropped: useType('counter'),
    TxInterests: useType('counter'),
    TxData: useType('counter'),
    TxNacks: useType('counter'),
    TxFrames: useType('counter'),
    TxOctets: useType('counter'),
  },
});

declareType('dpdk.EthStats', {
  type: 'object',
});

declareType('socketface.ExCounters', {
  type: 'object',
});

declareType('facemgmt.FaceInfo', {
  type: 'object',
  properties: {
    Id: useType('iface.FaceId'),
    LocalUri: useType('iface.FaceUri'),
    RemoteUri: useType('iface.FaceUri'),
    IsDown: useType('boolean'),
    Counters: useType('iface.Counters'),
    ExCounters: {
      oneOf: [
        useType('dpdk.EthStats'),
        useType('socketface.ExCounters'),
        true,
      ],
    },
    Latency: useType('running_stat.Snapshot'),
  },
});

declareType('facemgmt.IdArg', {
  type: 'object',
  properties: {
    Id: useType('iface.FaceId'),
  },
});

declareMethod('Face.List', 'null',
  {
    type: 'array',
    items: useType('iface.FaceId'),
    uniqueItems: true,
  });

declareMethod('Face.Get', 'facemgmt.IdArg', 'facemgmt.FaceInfo');

declareMethod('Face.Create',
  {
    type: 'object',
    properties: {
      LocalUri: useType('iface.FaceUri'),
      RemoteUri: useType('iface.FaceUri'),
    },
  },
  'facemgmt.IdArg');

declareMethod('Face.Destroy', 'facemgmt.IdArg', 'null');

};
})(exports);
