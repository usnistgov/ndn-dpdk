(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareType('net.HardwareAddr', {
  type: 'string',
});

ctx.declareType('iface.FaceUri', {
  type: 'string',
  format: 'uri',
});

ctx.declareType('ethface.Locator', ctx.markAllRequired({
  type: 'object',
  properties: {
    Scheme: { const: 'ether' },
    Port: {
      type: 'string',
    },
    Local: ctx.useType('net.HardwareAddr'),
    Remote: ctx.useType('net.HardwareAddr'),
  },
}));

ctx.declareType('socketface.Locator', {
  type: 'object',
  properties: {
    Scheme: {
      oneOf: [
        { const: 'udp' },
        { const: 'unixgram' },
        { const: 'tcp' },
        { const: 'unix' },
      ],
    },
    Local: {
      type: 'string',
    },
    Remote: {
      type: 'string',
    },
  },
  required: ['Scheme', 'Remote'],
});

ctx.declareType('iface.Locator', {
  oneOf: [
    ctx.useType('ethface.Locator'),
    ctx.useType('socketface.Locator'),
  ],
});

ctx.declareType('facemgmt.IdArg', ctx.markAllRequired({
  type: 'object',
  properties: {
    Id: ctx.useType('iface.FaceId'),
  },
}));

ctx.declareType('facemgmt.localRemoteUris', {
  type: 'object',
  properties: {
    LocalUri: ctx.useType('iface.FaceUri'),
    RemoteUri: ctx.useType('iface.FaceUri'),
  },
  required: ['RemoteUri'],
});

ctx.declareType('facemgmt.BasicInfo', {
  allOf: [
    ctx.useType('facemgmt.IdArg'),
    {
      properties: {
        Locator: ctx.useType('iface.Locator'),
      },
    },
  ],
});

ctx.declareType('facemgmt.BasicInfo[]', {
  type: 'array',
  items: ctx.useType('facemgmt.BasicInfo'),
});

ctx.declareType('iface.InOrderReassemblerCounters', {
  type: 'object',
  properties: {
    Accepted: ctx.useType('counter'),
    OutOfOrder: ctx.useType('counter'),
    Delivered: ctx.useType('counter'),
    Incomplete: ctx.useType('counter'),
  },
});

ctx.declareType('iface.Counters', {
  type: 'object',
  properties: {
    RxFrames: ctx.useType('counter'),
    RxOctets: ctx.useType('counter'),
    L2DecodeErrs: ctx.useType('counter'),
    Reass: ctx.useType('iface.InOrderReassemblerCounters'),
    L3DecodeErrs: ctx.useType('counter'),
    RxInterests: ctx.useType('counter'),
    RxData: ctx.useType('counter'),
    RxNacks: ctx.useType('counter'),
    FragGood: ctx.useType('counter'),
    FragBad: ctx.useType('counter'),
    TxAllocErrs: ctx.useType('counter'),
    TxQueued: ctx.useType('counter'),
    TxDropped: ctx.useType('counter'),
    TxInterests: ctx.useType('counter'),
    TxData: ctx.useType('counter'),
    TxNacks: ctx.useType('counter'),
    TxFrames: ctx.useType('counter'),
    TxOctets: ctx.useType('counter'),
  },
});

ctx.declareType('dpdk.EthStats', {
  type: 'object',
});

ctx.declareType('socketface.ExCounters', {
  type: 'object',
});

ctx.declareType('facemgmt.extendFaceInfo', {
  type: 'object',
  properties: {
    IsDown: ctx.useType('boolean'),
    Counters: ctx.useType('iface.Counters'),
    ExCounters: {
      oneOf: [
        ctx.useType('dpdk.EthStats'),
        ctx.useType('socketface.ExCounters'),
        true,
      ],
    },
    Latency: ctx.useType('running_stat.Snapshot'),
  },
});

ctx.declareType('facemgmt.FaceInfo', {
  allOf: [
    ctx.useType('facemgmt.BasicInfo'),
    ctx.useType('facemgmt.extendFaceInfo'),
  ],
});

ctx.declareMethod('Face.List', true, 'facemgmt.BasicInfo[]');

ctx.declareMethod('Face.Get', 'facemgmt.IdArg', 'facemgmt.FaceInfo');

ctx.declareMethod('Face.Create',
  {
    type: 'array',
    items: ctx.useType('facemgmt.localRemoteUris'),
    uniqueItems: true,
  },
  'facemgmt.BasicInfo[]');

ctx.declareMethod('Face.Destroy', 'facemgmt.IdArg', true);

};
})(exports);
