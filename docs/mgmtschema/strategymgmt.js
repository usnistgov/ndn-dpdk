(function(exports){
exports.provideDefinitions = function(ctx) {

ctx.declareType('strategymgmt.IdArg', ctx.markAllRequired({
  type: 'object',
  properties: {
    Id: ctx.useType('strategycode.Id'),
  },
}));

ctx.declareType('strategymgmt.StrategyInfo', ctx.markAllRequired({
  type: 'object',
  properties: {
    Id: ctx.useType('strategycode.Id'),
    Name: ctx.useType('string'),
  },
}));

ctx.declareMethod('Strategy.List', true,
  {
    type: 'array',
    items: ctx.useType('strategymgmt.StrategyInfo'),
  });

ctx.declareMethod('Strategy.Get', 'strategymgmt.IdArg', 'strategymgmt.StrategyInfo');

ctx.declareMethod('Strategy.Load',
  {
    type: 'object',
    properties: {
      Name: ctx.useType('string'),
      Elf: ctx.useType('blob'),
    },
    required: ['Elf'],
  }, 'strategymgmt.StrategyInfo');

ctx.declareMethod('Strategy.Unload', 'strategymgmt.IdArg', 'strategymgmt.StrategyInfo');

};
})(exports);
