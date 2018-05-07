(function(exports){
exports.provideDefinitions = function(declareType, useType, declareMethod) {

declareType('counter', {
  type: 'integer',
  minimum: 0,
})

declareType('running_stat.Snapshot', {
  type: 'object',
});

declareType('iface.FaceId', {
  type: 'integer',
  minimum: 1,
  maximum: 65535,
});

declareType('array-of_iface.FaceId', {
  type: 'array',
  items: useType('iface.FaceId'),
  minItems: 1,
  uniqueItems: true,
});

declareType('ndn.Name', {
  type: 'string',
  format: 'uri-reference',
});

};
})(exports);
