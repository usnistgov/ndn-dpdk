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

function declareType(type, subschema) {
  theSchema.definitions[type] = subschema;
}

function useType(t) {
  if (typeof t == 'object') {
    return t;
  }
  if (['null', 'boolean', 'array', 'number', 'string', 'integer'].includes(t)) {
    return {type: t};
  }
  console.assert(theSchema.definitions.hasOwnProperty(t), 'undefined type %s', t);
  return {'$ref': '#/definitions/' + t};
}

function declareMethod(method, paramsType, resultType) {
  theSchema.oneOf.push({
    properties: {
      method: {
        const: method,
      },
      params: useType(paramsType),
      result: useType(resultType),
    }
  });
}

['commontypes', 'facemgmt', 'fibmgmt', 'versionmgmt'].forEach(function(module) {
  require('./' + module).provideDefinitions(declareType, useType, declareMethod);
});

process.stdout.write(JSON.stringify(theSchema, null, 2));
