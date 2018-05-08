var theSchema = {
  '$schema': 'http://json-schema.org/draft-07/schema#',
  title: 'NDN-DPDK management API',
  definitions: {},
  type: 'object',
  properties: {
    method: {
      type: 'string'
    },
    params: true,
    result: true,
  },
  required: ['method'],
  additionalProperties: false,
  oneOf: [],
};

var ctx = {
  declareType: function(type, subschema) {
    theSchema.definitions[type] = subschema;
  },

  useType: function(t) {
    if (typeof t == 'object') {
      return t;
    }
    if (['null', 'boolean', 'array', 'number', 'string', 'integer'].includes(t)) {
      return {type: t};
    }
    console.assert(theSchema.definitions.hasOwnProperty(t), 'undefined type %s', t);
    return {'$ref': '#/definitions/' + t};
  },

  markAllRequired: function(subschema) {
    if (subschema.properties) {
      subschema.required = Object.keys(subschema.properties);
    }
    return subschema;
  },

  declareMethod: function(method, paramsType, resultType) {
    theSchema.oneOf.push({
      properties: {
        method: {
          const: method,
        },
        params: ctx.useType(paramsType),
        result: ctx.useType(resultType),
      }
    });
  },
};

['commontypes', 'facemgmt', 'fibmgmt', 'fwdpmgmt', 'ndtmgmt', 'versionmgmt'].forEach(function(module) {
  require('./' + module).provideDefinitions(ctx);
});

process.stdout.write(JSON.stringify(theSchema, null, 2));
